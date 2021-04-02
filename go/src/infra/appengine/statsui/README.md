# Infra Stats UI

Project to display stats for infrastructure performance

## Backend

Running the server:

```sh
# Needs to be run the first time to set up BigQuery credentials
gcloud auth application-default login
go run main.go
```

This will set up the backend server running on port `8800`
