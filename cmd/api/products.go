package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/martinezmoises/Test1/internal/data"
	"github.com/martinezmoises/Test1/internal/validator"
)

// Handler to create a new product
func (a *applicationDependencies) createProductHandler(w http.ResponseWriter, r *http.Request) {
	// Define a struct to hold the input data from the request body
	var input struct {
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		Category      string  `json:"category"`
		Price         float64 `json:"price"`
		ImageURL      string  `json:"image_url"`
		AverageRating float64 `json:"average_rating,omitempty"` // Optional field for initial rating
	}

	// Read and decode the JSON body into the input struct
	err := a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Create a new Product struct with the input data
	product := &data.Product{
		Name:          input.Name,
		Description:   input.Description,
		Category:      input.Category,
		Price:         input.Price,
		ImageURL:      input.ImageURL,
		AverageRating: input.AverageRating, // Initialize with the provided rating
	}

	// Initialize a validator and validate the product data
	v := validator.New()
	data.ValidateProduct(v, product)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert the new product into the database
	err = a.productModel.Insert(product)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Set the Location header for the newly created product and respond with JSON
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/products/%d", product.ID))
	data := envelope{"product": product}
	err = a.writeJSON(w, http.StatusCreated, data, headers)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to display a specific product by ID
func (a *applicationDependencies) displayProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the product ID from the URL and handle errors
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Retrieve the product from the database by ID
	product, err := a.productModel.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			a.notFoundResponse(w, r)
		default:
			a.serverErrorResponse(w, r, err)
		}
		return
	}

	// Respond with the product data in JSON format
	data := envelope{"product": product}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to update a specific product by ID
func (a *applicationDependencies) updateProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the product ID from the URL and handle errors
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Retrieve the existing product from the database
	product, err := a.productModel.Get(id)
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
		Name          *string  `json:"name"`
		Description   *string  `json:"description"`
		Category      *string  `json:"category"`
		Price         *float64 `json:"price"`
		ImageURL      *string  `json:"image_url"`
		AverageRating *float64 `json:"average_rating,omitempty"` // Optional field to update the rating
	}
	err = a.readJSON(w, r, &input)
	if err != nil {
		a.badRequestResponse(w, r, err)
		return
	}

	// Update the product fields if they are provided
	if input.Name != nil {
		product.Name = *input.Name
	}
	if input.Description != nil {
		product.Description = *input.Description
	}
	if input.Category != nil {
		product.Category = *input.Category
	}
	if input.Price != nil {
		product.Price = *input.Price
	}
	if input.ImageURL != nil {
		product.ImageURL = *input.ImageURL
	}
	if input.AverageRating != nil {
		product.AverageRating = *input.AverageRating
	}

	// Validate the updated product data
	v := validator.New()
	data.ValidateProduct(v, product)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update the product in the database
	err = a.productModel.Update(product)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the updated product data in JSON format
	data := envelope{"product": product}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to delete a specific product by ID
func (a *applicationDependencies) deleteProductHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the product ID from the URL and handle errors
	id, err := a.readIDParam(r)
	if err != nil {
		a.notFoundResponse(w, r)
		return
	}

	// Delete the product from the database
	err = a.productModel.Delete(id)
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
	data := envelope{"message": "product successfully deleted"}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}

// Handler to list all products with filtering, sorting, and pagination
func (a *applicationDependencies) listProductsHandler(w http.ResponseWriter, r *http.Request) {
	// Define a struct to hold query parameters for filtering and pagination
	var input struct {
		Name     string
		Category string
		data.Filters
	}

	// Parse query parameters from the URL
	queryParameters := r.URL.Query()
	input.Name = a.getSingleQueryParameter(queryParameters, "name", "")
	input.Category = a.getSingleQueryParameter(queryParameters, "category", "")

	// Initialize a validator and parse pagination/sorting parameters
	v := validator.New()
	input.Filters.Page = a.getSingleIntegerParameter(queryParameters, "page", 1, v)
	input.Filters.PageSize = a.getSingleIntegerParameter(queryParameters, "page_size", 10, v)
	input.Filters.Sort = a.getSingleQueryParameter(queryParameters, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "name", "-id", "-name", "price", "-price", "average_rating", "-average_rating"}

	// Validate filters and handle errors if necessary
	data.ValidateFilters(v, input.Filters)
	if !v.IsEmpty() {
		a.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Retrieve the list of products with the specified filters
	products, metadata, err := a.productModel.GetAll(input.Name, input.Category, input.Filters)
	if err != nil {
		a.serverErrorResponse(w, r, err)
		return
	}

	// Respond with the list of products and pagination metadata in JSON format
	data := envelope{"products": products, "@metadata": metadata}
	err = a.writeJSON(w, http.StatusOK, data, nil)
	if err != nil {
		a.serverErrorResponse(w, r, err)
	}
}
