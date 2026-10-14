// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	authv1 "k8s.io/api/authentication/v1"
	authzv1 "k8s.io/api/authorization/v1"
	corev1 "k8s.io/api/core/v1"
	flowv1 "k8s.io/api/flowcontrol/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/flowcontrol"
)

const ns = "kfp-apf-experiment"

type observation struct {
	mu         sync.Mutex
	Codes      map[int]int
	Schemas    map[string]int
	Priorities map[string]int
	WaitMS     float64
	WaitCalls  int
}
type measuredLimiter struct {
	flowcontrol.RateLimiter
	obs *observation
}

func (l measuredLimiter) Wait(ctx context.Context) error {
	start := time.Now()
	e := l.RateLimiter.Wait(ctx)
	l.obs.mu.Lock()
	l.obs.WaitMS += float64(time.Since(start).Microseconds()) / 1000
	l.obs.WaitCalls++
	l.obs.mu.Unlock()
	return e
}

type transport struct {
	base http.RoundTripper
	obs  *observation
}

func (t transport) RoundTrip(r *http.Request) (*http.Response, error) {
	v, e := t.base.RoundTrip(r)
	if v != nil {
		t.obs.mu.Lock()
		t.obs.Codes[v.StatusCode]++
		t.obs.Schemas[v.Header.Get("X-Kubernetes-PF-FlowSchema-UID")]++
		t.obs.Priorities[v.Header.Get("X-Kubernetes-PF-PriorityLevel-UID")]++
		t.obs.mu.Unlock()
	}
	return v, e
}

type sample struct {
	Name                  string
	QPS                   float32
	Workers               int
	Completed             int
	Errors                int
	WindowCanceled        int
	RequestTimeouts       int
	Throttled             int
	UnexpectedErrors      int
	AuthorizationFailures int
	DurationSeconds       float64
	P50MS                 float64
	P95MS                 float64
	MaxMS                 float64
	LimiterWaitMS         float64
	LimiterWaitCalls      int
	HTTP                  map[int]int
	FlowSchemas           map[string]int
	PriorityLevels        map[string]int
}
type actor struct {
	token                  string
	tokenClient, sarClient *kubernetes.Clientset
	obs                    *observation
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
func ptr[T any](v T) *T { return &v }
func newActor(base *rest.Config, token string, qps float32, burst int) actor {
	c := rest.CopyConfig(base)
	c.CertData = nil
	c.KeyData = nil
	c.CertFile = ""
	c.KeyFile = ""
	c.Username = ""
	c.Password = ""
	c.BearerToken = token
	c.BearerTokenFile = ""
	c.ExecProvider = nil
	c.AuthProvider = nil
	c.QPS = qps
	c.Burst = burst
	c.RateLimiter = nil
	c.Timeout = 5 * time.Second
	o := &observation{Codes: map[int]int{}, Schemas: map[string]int{}, Priorities: map[string]int{}}
	c.WrapTransport = func(rt http.RoundTripper) http.RoundTripper { return transport{rt, o} }
	if qps > 0 {
		c.RateLimiter = measuredLimiter{flowcontrol.NewTokenBucketRateLimiter(qps, burst), o}
	}
	a, e := kubernetes.NewForConfig(c)
	must(e)
	if qps > 0 {
		c.RateLimiter = measuredLimiter{flowcontrol.NewTokenBucketRateLimiter(qps, burst), o}
	}
	b, e := kubernetes.NewForConfig(c)
	must(e)
	return actor{token, a, b, o}
}
func run(ctx context.Context, name string, a actor, qps float32, workers int, duration time.Duration) sample {
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	start := time.Now()
	var mu sync.Mutex
	var lat []float64
	errorCount, authFailures := 0, 0
	windowCanceled, requestTimeouts, throttled, unexpected := 0, 0, 0, 0
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for ctx.Err() == nil {
				call, stop := context.WithTimeout(ctx, 3*time.Second)
				began := time.Now()
				tr, e := a.tokenClient.AuthenticationV1().TokenReviews().Create(call, &authv1.TokenReview{Spec: authv1.TokenReviewSpec{Token: a.token}}, metav1.CreateOptions{})
				bad := false
				if e == nil {
					bad = !tr.Status.Authenticated
					sr, se := a.sarClient.AuthorizationV1().SubjectAccessReviews().Create(call, &authzv1.SubjectAccessReview{Spec: authzv1.SubjectAccessReviewSpec{User: "system:serviceaccount:" + ns + ":load", Groups: []string{"system:serviceaccounts", "system:serviceaccounts:" + ns, "system:authenticated"}, ResourceAttributes: &authzv1.ResourceAttributes{Namespace: ns, Verb: "get", Resource: "configmaps"}}}, metav1.CreateOptions{})
					e = se
					if e == nil {
						bad = bad || !sr.Status.Allowed
					}
				}
				stop()
				mu.Lock()
				if e != nil {
					errorCount++
					switch {
					case ctx.Err() != nil:
						windowCanceled++
					case errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled) || strings.Contains(e.Error(), "would exceed context deadline"):
						requestTimeouts++
					case apierrors.IsTooManyRequests(e):
						throttled++
					default:
						unexpected++
					}
				} else {
					lat = append(lat, float64(time.Since(began).Microseconds())/1000)
				}
				if bad {
					authFailures++
				}
				mu.Unlock()
				if e != nil {
					// Bound generator retries after terminal errors without changing
					// client-go's HTTP retry or Retry-After handling.
					timer := time.NewTimer(10 * time.Millisecond)
					select {
					case <-ctx.Done():
						timer.Stop()
						return
					case <-timer.C:
					}
				}
			}
		}()
	}
	wg.Wait()
	sort.Float64s(lat)
	s := sample{Name: name, QPS: qps, Workers: workers, Completed: len(lat), Errors: errorCount, WindowCanceled: windowCanceled, RequestTimeouts: requestTimeouts, Throttled: throttled, UnexpectedErrors: unexpected, AuthorizationFailures: authFailures, DurationSeconds: time.Since(start).Seconds(), HTTP: a.obs.Codes, FlowSchemas: a.obs.Schemas, PriorityLevels: a.obs.Priorities, LimiterWaitMS: a.obs.WaitMS, LimiterWaitCalls: a.obs.WaitCalls}
	if len(lat) > 0 {
		s.P50MS = lat[len(lat)/2]
		s.P95MS = lat[(len(lat)-1)*95/100]
		s.MaxMS = lat[len(lat)-1]
	}
	return s
}

// metricTotal reads only the named APF series scoped to this experiment's
// jail priority. Counters are compared as deltas so stale observations cannot
// satisfy the queueing assertion.
func metricTotal(data []byte, metric string, labels ...string) float64 {
	total := 0.0
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, metric+"{") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		matches := strings.Contains(fields[0], `priority_level="`+ns+`-jail"`)
		for _, label := range labels {
			matches = matches && strings.Contains(fields[0], label)
		}
		if !matches {
			continue
		}
		value, err := strconv.ParseFloat(fields[1], 64)
		if err == nil {
			total += value
		}
	}
	return total
}
func queuedDuring(before, after []byte) bool {
	return metricTotal(after, "apiserver_flowcontrol_rejected_requests_total", `reason="time-out"`) > metricTotal(before, "apiserver_flowcontrol_rejected_requests_total", `reason="time-out"`) &&
		metricTotal(after, "apiserver_flowcontrol_request_wait_duration_seconds_sum") > metricTotal(before, "apiserver_flowcontrol_request_wait_duration_seconds_sum") &&
		metricTotal(after, "apiserver_flowcontrol_request_wait_duration_seconds_count") > metricTotal(before, "apiserver_flowcontrol_request_wait_duration_seconds_count")
}

func main() {
	kubeconfig := flag.String("kubeconfig", "", "Dedicated Kind kubeconfig (required)")
	out := flag.String("out", "apf-results", "Output directory")
	seconds := flag.Int("seconds", 10, "Seconds per sample")
	flag.Parse()
	if *seconds < 1 || *seconds > 30 {
		panic("seconds must be between1 and30")
	}
	if *kubeconfig == "" {
		panic("explicit kubeconfig required")
	}
	must(os.MkdirAll(*out, 0700))
	raw, e := clientcmd.LoadFromFile(*kubeconfig)
	must(e)
	if raw.CurrentContext != "kind-kfp-apf-experiment" {
		panic("refusing context other than kind-kfp-apf-experiment")
	}
	cfg, e := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	must(e)
	cfg.QPS = -1
	admin, e := kubernetes.NewForConfig(cfg)
	must(e)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	_, e = admin.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
	must(e)
	defer func() {
		cleanup, c := context.WithTimeout(context.Background(), 20*time.Second)
		defer c()
		for _, n := range []string{"load", "control"} {
			_ = admin.FlowcontrolV1().FlowSchemas().Delete(cleanup, ns+"-"+n, metav1.DeleteOptions{})
		}
		for _, n := range []string{"load", "control", "jail"} {
			_ = admin.FlowcontrolV1().PriorityLevelConfigurations().Delete(cleanup, ns+"-"+n, metav1.DeleteOptions{})
		}
		_ = admin.RbacV1().ClusterRoleBindings().Delete(cleanup, ns, metav1.DeleteOptions{})
		_ = admin.RbacV1().ClusterRoles().Delete(cleanup, ns, metav1.DeleteOptions{})
		_ = admin.CoreV1().Namespaces().Delete(cleanup, ns, metav1.DeleteOptions{})
	}()
	tokens := map[string]string{}
	for _, n := range []string{"load", "control"} {
		_, e = admin.CoreV1().ServiceAccounts(ns).Create(ctx, &corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: n}}, metav1.CreateOptions{})
		must(e)
		tr, err := admin.CoreV1().ServiceAccounts(ns).CreateToken(ctx, n, &authv1.TokenRequest{Spec: authv1.TokenRequestSpec{ExpirationSeconds: ptr(int64(600))}}, metav1.CreateOptions{})
		must(err)
		tokens[n] = tr.Status.Token
	}
	_, e = admin.RbacV1().ClusterRoles().Create(ctx, &rbacv1.ClusterRole{ObjectMeta: metav1.ObjectMeta{Name: ns}, Rules: []rbacv1.PolicyRule{{APIGroups: []string{"authentication.k8s.io"}, Resources: []string{"tokenreviews"}, Verbs: []string{"create"}}, {APIGroups: []string{"authorization.k8s.io"}, Resources: []string{"subjectaccessreviews"}, Verbs: []string{"create"}}}}, metav1.CreateOptions{})
	must(e)
	_, e = admin.RbacV1().ClusterRoleBindings().Create(ctx, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: ns}, RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "ClusterRole", Name: ns}, Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "load", Namespace: ns}, {Kind: "ServiceAccount", Name: "control", Namespace: ns}}}, metav1.CreateOptions{})
	must(e)
	_, e = admin.RbacV1().Roles(ns).Create(ctx, &rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: "read"}, Rules: []rbacv1.PolicyRule{{APIGroups: []string{""}, Resources: []string{"configmaps"}, Verbs: []string{"get"}}}}, metav1.CreateOptions{})
	must(e)
	_, e = admin.RbacV1().RoleBindings(ns).Create(ctx, &rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "read"}, RoleRef: rbacv1.RoleRef{APIGroup: rbacv1.GroupName, Kind: "Role", Name: "read"}, Subjects: []rbacv1.Subject{{Kind: "ServiceAccount", Name: "load", Namespace: ns}}}, metav1.CreateOptions{})
	must(e)
	for n, shares := range map[string]int32{"load": 1, "control": 10, "jail": 0} {
		_, e = admin.FlowcontrolV1().PriorityLevelConfigurations().Create(ctx, &flowv1.PriorityLevelConfiguration{ObjectMeta: metav1.ObjectMeta{Name: ns + "-" + n}, Spec: flowv1.PriorityLevelConfigurationSpec{Type: flowv1.PriorityLevelEnablementLimited, Limited: &flowv1.LimitedPriorityLevelConfiguration{NominalConcurrencyShares: ptr(shares), LendablePercent: ptr(int32(0)), BorrowingLimitPercent: ptr(int32(0)), LimitResponse: flowv1.LimitResponse{Type: flowv1.LimitResponseTypeQueue, Queuing: &flowv1.QueuingConfiguration{Queues: 1, HandSize: 1, QueueLengthLimit: 2}}}}}, metav1.CreateOptions{})
		must(e)
	}
	for _, n := range []string{"load", "control"} {
		_, e = admin.FlowcontrolV1().FlowSchemas().Create(ctx, &flowv1.FlowSchema{ObjectMeta: metav1.ObjectMeta{Name: ns + "-" + n}, Spec: flowv1.FlowSchemaSpec{MatchingPrecedence: 2, PriorityLevelConfiguration: flowv1.PriorityLevelConfigurationReference{Name: ns + "-" + n}, DistinguisherMethod: &flowv1.FlowDistinguisherMethod{Type: flowv1.FlowDistinguisherMethodByUserType}, Rules: []flowv1.PolicyRulesWithSubjects{{Subjects: []flowv1.Subject{{Kind: flowv1.SubjectKindServiceAccount, ServiceAccount: &flowv1.ServiceAccountSubject{Namespace: ns, Name: n}}}, ResourceRules: []flowv1.ResourcePolicyRule{{Verbs: []string{"create"}, APIGroups: []string{"authentication.k8s.io", "authorization.k8s.io"}, Resources: []string{"tokenreviews", "subjectaccessreviews"}, ClusterScope: true}}}}}}, metav1.CreateOptions{})
		must(e)
	}
	time.Sleep(3 * time.Second)
	probe := newActor(cfg, tokens["load"], -1, 40)
	denied, err := probe.sarClient.AuthorizationV1().SubjectAccessReviews().Create(ctx, &authzv1.SubjectAccessReview{Spec: authzv1.SubjectAccessReviewSpec{User: "system:serviceaccount:" + ns + ":load", ResourceAttributes: &authzv1.ResourceAttributes{Namespace: "kube-system", Verb: "get", Resource: "secrets"}}}, metav1.CreateOptions{})
	must(err)
	if denied.Status.Allowed {
		panic("cross-namespace secret read unexpectedly allowed")
	}
	snapshot := func(name string) []byte {
		b, err := admin.RESTClient().Get().AbsPath("/metrics").DoRaw(ctx)
		must(err)
		if len(b) > 16<<20 {
			panic("metrics too large")
		}
		must(os.WriteFile(filepath.Join(*out, name+"-metrics.txt"), b, 0600))
		return b
	}
	fs, e := admin.FlowcontrolV1().FlowSchemas().List(ctx, metav1.ListOptions{})
	must(e)
	pl, e := admin.FlowcontrolV1().PriorityLevelConfigurations().List(ctx, metav1.ListOptions{})
	must(e)
	for n, v := range map[string]any{"flowschemas": fs, "prioritylevels": pl} {
		b, err := json.MarshalIndent(v, "", "  ")
		must(err)
		must(os.WriteFile(filepath.Join(*out, n+".json"), b, 0600))
	}
	results := []sample{}
	persistResults := func() {
		b, err := json.MarshalIndent(results, "", "  ")
		must(err)
		temporary := filepath.Join(*out, "results.json.tmp")
		must(os.WriteFile(temporary, b, 0600))
		must(os.Rename(temporary, filepath.Join(*out, "results.json")))
	}
	duration := time.Duration(*seconds) * time.Second
	for _, setting := range []struct {
		q float32
		b int
	}{{5, 10}, {20, 40}, {-1, 40}} {
		for _, workers := range []int{1, 32} {
			name := fmt.Sprintf("qps%g-workers%d", setting.q, workers)
			snapshot(name + "-before")
			a := newActor(cfg, tokens["load"], setting.q, setting.b)
			control := newActor(cfg, tokens["control"], -1, 40)
			ch := make(chan sample, 1)
			go func() { ch <- run(ctx, name+"-control", control, -1, 1, duration) }()
			results = append(results, run(ctx, name, a, setting.q, workers, duration), <-ch)
			persistResults()
			snapshot(name + "-after")
		}
	}
	jail, e := admin.FlowcontrolV1().FlowSchemas().Get(ctx, ns+"-load", metav1.GetOptions{})
	must(e)
	jail.Spec.PriorityLevelConfiguration.Name = ns + "-jail"
	_, e = admin.FlowcontrolV1().FlowSchemas().Update(ctx, jail, metav1.UpdateOptions{})
	must(e)
	time.Sleep(3 * time.Second)
	beforeJail := snapshot("jail-before")
	jailResult := make(chan sample, 1)
	go func() {
		jailResult <- run(ctx, "synthetic-zero-seat-cancel-retry", newActor(cfg, tokens["load"], -1, 40), -1, 16, 6*time.Second)
	}()
	time.Sleep(2 * time.Second)
	snapshot("jail-during")
	results = append(results, <-jailResult)
	persistResults()
	afterJail := snapshot("jail-after")
	queued := queuedDuring(beforeJail, afterJail)
	jail, e = admin.FlowcontrolV1().FlowSchemas().Get(ctx, ns+"-load", metav1.GetOptions{})
	must(e)
	jail.Spec.PriorityLevelConfiguration.Name = ns + "-load"
	_, e = admin.FlowcontrolV1().FlowSchemas().Update(ctx, jail, metav1.UpdateOptions{})
	must(e)
	time.Sleep(3 * time.Second)
	results = append(results, run(ctx, "recovery", newActor(cfg, tokens["load"], -1, 40), -1, 1, 3*time.Second))
	persistResults()
	b, e := json.MarshalIndent(results, "", "  ")
	must(e)
	must(os.WriteFile(filepath.Join(*out, "results.json"), b, 0600))
	fmt.Println(string(b))
	schemaIDs := map[string]string{}
	priorityIDs := map[string]string{}
	for _, f := range fs.Items {
		schemaIDs[f.Name] = string(f.UID)
	}
	for _, p := range pl.Items {
		priorityIDs[p.Name] = string(p.UID)
	}
	if !queued {
		panic("synthetic phase did not increase APF queue-timeout and wait metrics")
	}
	valid := true
	for _, s := range results {
		if s.AuthorizationFailures != 0 || s.UnexpectedErrors != 0 {
			valid = false
		}
		if !strings.HasPrefix(s.Name, "synthetic") && s.Completed == 0 {
			valid = false
		}
		identity := "load"
		if strings.HasSuffix(s.Name, "-control") {
			identity = "control"
		}
		priority := identity
		if strings.HasPrefix(s.Name, "synthetic") {
			priority = "jail"
			if s.Completed != 0 || s.HTTP[429] == 0 {
				valid = false
			}
		}
		for uid := range s.FlowSchemas {
			if uid != schemaIDs[ns+"-"+identity] {
				valid = false
			}
		}
		for uid := range s.PriorityLevels {
			if uid != priorityIDs[ns+"-"+priority] {
				valid = false
			}
		}
		if s.FlowSchemas[""] > 0 || s.PriorityLevels[""] > 0 {
			valid = false
		}
	}
	if !valid {
		panic("authentication/authorization, completion, or APF-header validation failed")
	}
}
