variable "TAG" {
  default = "latest"
}

variable "REGISTRY" {
  default = "kind-registry:5000"
}

variable "KFP_REPO" {
  default = "."
}

variable "PLATFORM" {
  default = "linux/amd64"
}

group "default" {
  targets = [
    "api-server",
    "driver",
    "launcher",
    "persistence-agent",
    "scheduled-workflow"
  ]
}

target "api-server" {
  context = "${KFP_REPO}"
  dockerfile = "backend/Dockerfile"
  tags = ["${REGISTRY}/apiserver:${TAG}"]
  platforms = ["${PLATFORM}"]
}

target "driver" {
  context = "${KFP_REPO}"
  dockerfile = "backend/Dockerfile.driver"
  tags = ["${REGISTRY}/driver:${TAG}"]
  platforms = ["${PLATFORM}"]
}

target "launcher" {
  context = "${KFP_REPO}"
  dockerfile = "backend/Dockerfile.launcher"
  tags = ["${REGISTRY}/launcher:${TAG}"]
  platforms = ["${PLATFORM}"]
}

target "persistence-agent" {
  context = "${KFP_REPO}"
  dockerfile = "backend/Dockerfile.persistenceagent"
  tags = ["${REGISTRY}/persistenceagent:${TAG}"]
  platforms = ["${PLATFORM}"]
}

target "scheduled-workflow" {
  context = "${KFP_REPO}"
  dockerfile = "backend/Dockerfile.scheduledworkflow"
  tags = ["${REGISTRY}/scheduledworkflow:${TAG}"]
  platforms = ["${PLATFORM}"]
}