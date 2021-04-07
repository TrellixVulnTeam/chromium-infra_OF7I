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

## Frontend

Running the frontend:

```sh
cd frontend
npm install
npm start
```

This will set up the frontend client running on port `3000` with an automatic
proxy to the backend server running on `8800`.  To view the UI, go to
[localhost:3000](http://localhost:3000)

Formatting:

```sh
npm run fix
```

## Deployment

```sh
./deploy.sh
```

See the latest version at [https://chrome-infra-stats.googleplex.com/](https://chrome-infra-stats.googleplex.com/)

## [Roadmap](ROADMAP.md)
