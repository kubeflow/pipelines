// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package awsiam authenticates to RDS and Aurora with IAM database
// authentication. The password is a signed token that expires after fifteen
// minutes, so it is generated per connection rather than held in configuration.
package awsiam

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
)

// ProviderName identifies this provider in DB_CREDENTIAL_PROVIDER.
const ProviderName = "aws-iam"

// tokenTTL is how long a generated token is reused. RDS tokens are valid for
// fifteen minutes; reusing one for ten leaves room for clock skew and for a
// connection that is slow to complete its handshake.
const tokenTTL = 10 * time.Minute

// credentialExpiryMargin is how long before the signing credentials expire the
// cached token is abandoned. A token is a request signed with those
// credentials, so RDS stops accepting it when they expire, whatever its own
// fifteen-minute lifetime would allow. The margin covers clock skew and a
// handshake that begins just before the boundary.
const credentialExpiryMargin = time.Minute

func init() {
	dbcreds.RegisterFactory(factory{})
}

type factory struct{}

func (factory) Name() string { return ProviderName }

// RequiresTLS reports that this provider cannot operate over an unverified
// connection. RDS requires an encrypted connection for IAM authentication, and
// on MySQL the token is sent verbatim under the cleartext password plugin.
func (factory) RequiresTLS() bool { return true }

func (factory) New(cfg dbcreds.Config) (dbcreds.Provider, error) {
	// LoadDefaultConfig resolves IRSA, EKS Pod Identity and instance profiles
	// through the SDK's default chain, the same way the object store does. It
	// runs once at startup, so a background context is appropriate.
	var options []func(*config.LoadOptions) error
	if region := cfg.Setting(dbcreds.SettingRegion); region != "" {
		options = append(options, config.WithRegion(region))
	}
	awsConfig, err := config.LoadDefaultConfig(context.Background(), options...)
	if err != nil {
		return nil, fmt.Errorf("load AWS configuration: %w", err)
	}
	if awsConfig.Region == "" {
		return nil, fmt.Errorf(`no AWS region is configured; set DB_CREDENTIAL_PROVIDER_SETTINGS to {"region":"..."} or set AWS_REGION`)
	}
	if awsConfig.Credentials == nil {
		return nil, fmt.Errorf("no AWS credentials are available; give the pod's service account an IAM role")
	}

	return &provider{
		credentials: awsConfig.Credentials,
		region:      awsConfig.Region,
		now:         time.Now,
		tokens:      map[string]cachedToken{},
	}, nil
}

type cachedToken struct {
	token   string
	expires time.Time
}

type provider struct {
	credentials aws.CredentialsProvider
	region      string
	now         func() time.Time

	mu     sync.Mutex
	tokens map[string]cachedToken
}

func (p *provider) Name() string { return ProviderName }

func (p *provider) Connector(ctx context.Context, t dbcreds.Target) (driver.Connector, error) {
	// RDS requires an encrypted connection for IAM authentication, and on MySQL
	// the token is sent using the cleartext password plugin, so without TLS it
	// would cross the network in the clear.
	if t.TLS == nil {
		return nil, fmt.Errorf("IAM database authentication requires TLS; set DB_TLS_CA_PATH to the database CA bundle")
	}

	switch t.Driver {
	case dbcreds.DriverMySQL:
		return p.mysqlConnector(t)
	case dbcreds.DriverPostgreSQL:
		return p.postgresConnector(t)
	default:
		return nil, dbcreds.UnsupportedDriverError(ProviderName, t.Driver)
	}
}

func (p *provider) mysqlConnector(t dbcreds.Target) (driver.Connector, error) {
	config, err := dbcreds.MySQLConfig(t)
	if err != nil {
		return nil, err
	}
	// Defense in depth: MySQLConfig rejects a configuration that permits a
	// plaintext fallback, and the cleartext plugin must never be armed on a
	// connection that could take one, because it sends the token verbatim.
	if config.AllowFallbackToPlaintext {
		return nil, fmt.Errorf("refusing to enable the cleartext password plugin on a connection that permits a plaintext fallback")
	}
	config.AllowCleartextPasswords = true

	// BeforeConnect runs for every new connection with its own copy of the
	// configuration, which is what keeps expired tokens out of the pool.
	return dbcreds.MySQLConnector(config, mysql.BeforeConnect(
		func(ctx context.Context, connConfig *mysql.Config) error {
			token, err := p.token(ctx, connConfig.Addr, connConfig.User)
			if err != nil {
				return err
			}
			connConfig.Passwd = token
			return nil
		}))
}

func (p *provider) postgresConnector(t dbcreds.Target) (driver.Connector, error) {
	config, err := dbcreds.PostgreSQLConfig(t)
	if err != nil {
		return nil, err
	}
	if err := verifyRoutes(config); err != nil {
		return nil, err
	}

	// The pgx equivalent of BeforeConnect, likewise invoked per connection with
	// a copy of the configuration.
	return dbcreds.PostgreSQLConnector(config, stdlib.OptionBeforeConnect(
		func(ctx context.Context, connConfig *pgx.ConnConfig) error {
			endpoint := net.JoinHostPort(connConfig.Host, strconv.Itoa(int(connConfig.Port)))
			token, err := p.token(ctx, endpoint, connConfig.User)
			if err != nil {
				return err
			}
			connConfig.Password = token
			return nil
		})), nil
}

// verifyRoutes rejects a configuration whose connection attempts would not all
// reach the endpoint the token is signed for, over a verified connection.
//
// A token is a request signed for one host and port, and the hook below runs
// with the primary configuration only, so any fallback either goes somewhere
// the token does not cover or takes a route that drops TLS. A Unix socket does
// both: the driver never encrypts one, even under sslmode=verify-full, so the
// token would be handed to whoever owns the socket in the clear.
//
// Checking the resolved configuration rather than the requested sslmode is what
// makes this reliable -- sslmode is what was asked for, and these are the routes
// the driver will actually take.
func verifyRoutes(config *pgx.ConnConfig) error {
	if config.TLSConfig == nil {
		return fmt.Errorf("IAM database authentication requires a verified connection, but the PostgreSQL parameters resolve to an unencrypted route to %q; a Unix socket host cannot be used, because the driver does not encrypt one whatever sslmode asks for", config.Host)
	}
	// Any fallback at all is a problem, so only the first needs reporting.
	if len(config.Fallbacks) > 0 {
		fallback := config.Fallbacks[0]
		if fallback.TLSConfig == nil {
			return fmt.Errorf("IAM database authentication requires a verified connection, but the PostgreSQL parameters allow falling back to an unencrypted route to %q; remove it", fallback.Host)
		}
		return fmt.Errorf("IAM database authentication signs a token for one endpoint, but the PostgreSQL parameters resolve to more than one (%s:%d, then %s:%d); configure a single host, such as the cluster writer endpoint",
			config.Host, config.Port, fallback.Host, fallback.Port)
	}
	return nil
}

// token returns a valid authentication token for addr, generating one if the
// cached token is missing or close to expiry.
//
// The lock is held only to read and write the cache, never across generation.
// Signing the request is local, but it first retrieves the AWS credentials, and
// a cold or expiring cache makes that a call to STS. Holding the lock across it
// would block every other new connection behind one slow refresh, which surfaces
// as database latency rather than as credential latency. A burst during a
// refresh may sign more than once; that is cheaper than serializing the pool.
func (p *provider) token(ctx context.Context, addr, user string) (string, error) {
	key := addr + "|" + user

	p.mu.Lock()
	cached, ok := p.tokens[key]
	p.mu.Unlock()
	if ok && p.now().Before(cached.expires) {
		return cached.token, nil
	}

	// Retrieve the credentials here rather than letting BuildAuthToken do it,
	// so that the token can be cached against the lifetime of the credentials
	// that signed it. Temporary credentials -- which is what IRSA and Pod
	// Identity hand out -- can have less time left than the token would.
	credentials, err := p.credentials.Retrieve(ctx)
	if err != nil {
		return "", fmt.Errorf("retrieve AWS credentials to sign a database authentication token for user %q at %s: %w", user, addr, err)
	}
	signer := aws.CredentialsProviderFunc(func(context.Context) (aws.Credentials, error) {
		return credentials, nil
	})

	token, err := auth.BuildAuthToken(ctx, addr, p.region, user, signer)
	if err != nil {
		return "", fmt.Errorf("generate an RDS IAM authentication token for user %q at %s: %w", user, addr, err)
	}

	expires := p.now().Add(tokenTTL)
	if credentials.CanExpire {
		if limit := credentials.Expires.Add(-credentialExpiryMargin); limit.Before(expires) {
			expires = limit
		}
	}
	// Credentials already inside the margin produce a token that is usable now
	// but must not be reused, so it is returned without being cached and the
	// next connection retrieves again.
	if expires.After(p.now()) {
		p.mu.Lock()
		p.tokens[key] = cachedToken{token: token, expires: expires}
		p.mu.Unlock()
	}
	return token, nil
}
