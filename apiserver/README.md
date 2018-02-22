To start a webserver, run 

> docker build .

> docker run -p 8888:8888 -v ~/go/src/github.com/googleprivate/ml:/go/src/github.com/googleprivate/ml -t -i [image_id]


// TODO(yangpa): I'll use k8s deployment spec instead. And better test framework, CI, etc.  