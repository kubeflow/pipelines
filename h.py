Step 3/23 : RUN curl --silent --location "https://github.com/weaveworks/eksctl/releases/download/v0.134.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && chmod +x /tmp/eksctl && mv /tmp/eksctl /usr/local/bin/
 ---> Running in 0d1750f74989

gzip: stdin: unexpected end of file
tar: Child returned status 1
tar: Error is not recoverable: exiting now
The command '/bin/sh -c curl --silent --location "https://github.com/weaveworks/eksctl/releases/download/v0.134.0/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp && chmod +x /tmp/eksctl && mv /tmp/eksctl /usr/local/bin/' returned a non-zero code: 2