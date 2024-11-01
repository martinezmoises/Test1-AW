## run: run the cmd/api application

.PHONY : run
run:
	@echo 'Running application...'
	@go run ./cmd/api -port=4000 -env=development -db-dsn=${PRODUCTS_DB_DSN}

## db/psql: connect to the database using psql (terminal)
.PHONY: db/psql
db/psql:
	psql ${PRODUCTS_DB_DSN}


