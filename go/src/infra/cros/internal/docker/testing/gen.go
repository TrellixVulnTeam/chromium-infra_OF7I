package testing

//go:generate mockgen -destination client_mock.go -package testing github.com/docker/docker/client ContainerAPIClient
