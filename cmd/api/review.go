package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/martinezmoises/Test1/internal/data"
	"github.com/martinezmoises/Test1/internal/validator"
)

// Handler to create a review for a specific product
func (a *applicationDependencies) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the product ID from the URL and handle errors
	productID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Define a struct to hold the input data from the request body
	var input struct {
		Content string `json:"content"`
		Author  string `json:"author"`
		Rating  int    `json:"rating"`
	}

	// Read and decode the JSON body into the input struct
	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Create a new Review struct with the provided input data and product ID
	review := &data.Review{
		ProductID: productID,
		Content:   input.Content,
		Author:    input.Author,
		Rating:    input.Rating,
	}

	// Initialize a validator and validate the review data
	v := validator.New()
	data.ValidateReview(v, review)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert the new review into the database
	err = a.reviewModel.Insert(review)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Set the Location header for the newly created review and respond with JSON
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/products/%d/reviews/%d", productID, review.ID))
	data := envelope{"review": review}
	err = a.writeJSON(w, http.StatusCreated, data, headers)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to display a specific review for a specific product
func (a *applicationDependencies) displayReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Extract product ID and review ID from the URL
	productID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}
	reviewID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Retrieve the review from the database
	review, err := a.reviewModel.Get(productID, reviewID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Respond with the review data in JSON format
	data := envelope{"review": review}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to update a specific review for a specific product
func (a *applicationDependencies) updateReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Extract product ID and review ID from the URL
	productID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}
	reviewID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Retrieve the existing review from the database
	review, err := a.reviewModel.Get(productID, reviewID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Define a struct to hold optional fields for partial updates
	var input struct {
		Content      *string `json:"content"`
		Author       *string `json:"author"`
		Rating       *int    `json:"rating"`
		HelpfulCount *int    `json:"helpful_count"`
	}

	// Decode the JSON body into the input struct
	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Update the review fields if they are provided
	if input.Content != nil {
		review.Content = *input.Content
	}
	if input.Author != nil {
		review.Author = *input.Author
	}
	if input.Rating != nil {
		review.Rating = *input.Rating
	}
	if input.HelpfulCount != nil {
		review.HelpfulCount = *input.HelpfulCount
	}

	// Validate the updated review data
	v := validator.New()
	data.ValidateReview(v, review)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update the review in the database
	err = a.reviewModel.Update(review)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the updated review data in JSON format
	data := envelope{"review": review}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to delete a specific review for a specific product
func (a *applicationDependencies) deleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	// Extract product ID and review ID from the URL
	productID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}
	reviewID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Delete the review from the database
	err = a.reviewModel.Delete(productID, reviewID)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Respond with a success message in JSON format
	data := envelope{"message": "review successfully deleted"}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to list all reviews for a specific product
func (a *applicationDependencies) listReviewsForProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract product ID from the URL
	productID, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Parse query parameters for pagination and sorting
	var filters data.Filters
	queryParameters := r.URL.Query()
	v := validator.New()
	filters.Page = a.getSingleIntegerParameter(queryParameters, "page", 1, v)
	filters.PageSize = a.getSingleIntegerParameter(queryParameters, "page_size", 10, v)
	filters.Sort = a.getSingleQueryParameter(queryParameters, "sort", "id")
	filters.SortSafeList = []string{"id", "rating", "-id", "-rating"}

	// Validate filters and handle errors if necessary
	data.ValidateFilters(v, filters)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve the list of reviews for the product with pagination and sorting
	reviews, metadata, err := a.reviewModel.GetAll(productID, filters)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the list of reviews and pagination metadata in JSON format
	data := envelope{"reviews": reviews, "@metadata": metadata}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to list all reviews (not limited to a specific product)
func (a *applicationDependencies) listAllReviewsHandler(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for pagination and sorting
	var filters data.Filters
	queryParameters := r.URL.Query()
	v := validator.New()
	filters.Page = a.getSingleIntegerParameter(queryParameters, "page", 1, v)
	filters.PageSize = a.getSingleIntegerParameter(queryParameters, "page_size", 10, v)
	filters.Sort = a.getSingleQueryParameter(queryParameters, "sort", "id")
	filters.SortSafeList = []string{"id", "rating", "-id", "-rating"}

	// Validate filters and handle errors if necessary
	data.ValidateFilters(v, filters)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve all reviews with pagination and sorting, independent of product ID
	reviews, metadata, err := a.reviewModel.GetAll(0, filters) // Passing 0 to indicate no specific product
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the list of all reviews and pagination metadata in JSON format
	data := envelope{"reviews": reviews, "@metadata": metadata}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}
