package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (a *applicationDependencies) routes() http.Handler {

	// setup a new router
	router := httprouter.New()
	// handle 404
	router.NotFound = http.HandlerFunc(a.notFoundResponse)
	// handle 405
	router.MethodNotAllowed = http.HandlerFunc(a.methodNotAllowedResponse)
	// setup routes
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", a.healthCheckHandler)
	router.HandlerFunc(http.MethodPost, "/v1/products", a.createProductHandler)
	router.HandlerFunc(http.MethodGet, "/v1/products/:id", a.displayProductHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/products/:id", a.updateProductHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/products/:id", a.deleteProductHandler)
	router.HandlerFunc(http.MethodGet, "/v1/products", a.listProductsHandler)
	//Reviews Routes
	router.HandlerFunc(http.MethodPost, "/v1/products/:id/reviews", a.createReviewHandler)
	router.HandlerFunc(http.MethodGet, "/v1/products/:id/reviews", a.listReviewsForProductHandler)
	router.HandlerFunc(http.MethodGet, "/v1/products/:id/reviews/:review_id", a.displayReviewHandler)
	router.HandlerFunc(http.MethodPatch, "/v1/products/:id/reviews/:review_id", a.updateReviewHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/products/:id/reviews/:review_id", a.deleteReviewHandler)
	router.HandlerFunc(http.MethodGet, "/v1/reviews", a.listAllReviewsHandler)

	return a.recoverPanic(a.rateLimit(router))

}
