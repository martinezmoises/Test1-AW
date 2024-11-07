package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/martinezmoises/Test1/internal/validator"
)

var ErrRecordNotFound = errors.New("record not found")

// Product struct represents a product with various attributes.
type Product struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Category      string    `json:"category"`
	Price         float64   `json:"price"`
	ImageURL      string    `json:"image_url"`
	AverageRating float64   `json:"average_rating"`
	CreatedAt     time.Time `json:"-"`
	Version       int32     `json:"version"`
}

// ProductModel struct wraps the DB connection pool.
type ProductModel struct {
	DB *sql.DB
}

// ValidateProduct checks the fields of the Product struct.
func ValidateProduct(v *validator.Validator, product *Product) {
	v.Check(product.Name != "", "name", "must be provided")
	v.Check(len(product.Name) <= 100, "name", "must not be more than 100 characters long")
	v.Check(product.Description != "", "description", "must be provided")
	v.Check(len(product.Description) <= 500, "description", "must not be more than 500 characters long")
	v.Check(product.Category != "", "category", "must be provided")
	v.Check(product.Price > 0, "price", "must be a positive number")
	v.Check(len(product.ImageURL) <= 255, "image_url", "must not be more than 255 characters long")
	v.Check(product.AverageRating >= 0 && product.AverageRating <= 5, "average_rating", "must be between 0 and 5")
}

// Insert inserts a new product into the database and returns the created product ID, creation time, and version.
func (p ProductModel) Insert(product *Product) error {
	query := `
		INSERT INTO products (name, description, category, price, image_url, average_rating)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, version
	`
	args := []any{product.Name, product.Description, product.Category, product.Price, product.ImageURL, product.AverageRating}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return p.DB.QueryRowContext(ctx, query, args...).Scan(&product.ID, &product.CreatedAt, &product.Version)
}

// Get retrieves a specific product by ID.
func (p ProductModel) Get(id int64) (*Product, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
		SELECT id, created_at, name, description, category, price, image_url, average_rating, version
		FROM products
		WHERE id = $1
	`

	var product Product
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := p.DB.QueryRowContext(ctx, query, id).Scan(
		&product.ID,
		&product.CreatedAt,
		&product.Name,
		&product.Description,
		&product.Category,
		&product.Price,
		&product.ImageURL,
		&product.AverageRating,
		&product.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}
	return &product, nil
}

// Update modifies an existing product in the database.
func (p ProductModel) Update(product *Product) error {
	query := `
		UPDATE products
		SET name = $1, description = $2, category = $3, price = $4, image_url = $5, average_rating = $6, version = version + 1
		WHERE id = $7
		RETURNING version
	`
	args := []any{product.Name, product.Description, product.Category, product.Price, product.ImageURL, product.AverageRating, product.ID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return p.DB.QueryRowContext(ctx, query, args...).Scan(&product.Version)
}

// Delete removes a product from the database by ID.
func (p ProductModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
		DELETE FROM products
		WHERE id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := p.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// GetAll retrieves all products, with filtering, sorting, and pagination.
func (p ProductModel) GetAll(name, category string, filters Filters) ([]*Product, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) OVER(), id, created_at, name, description, category, price, image_url, average_rating, version
		FROM products
		WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
		AND (category = $2 OR $2 = '')
		ORDER BY %s %s, id ASC
		LIMIT $3 OFFSET $4`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := p.DB.QueryContext(ctx, query, name, category, filters.limit(), filters.offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	products := []*Product{}

	for rows.Next() {
		var product Product
		err := rows.Scan(
			&totalRecords,
			&product.ID,
			&product.CreatedAt,
			&product.Name,
			&product.Description,
			&product.Category,
			&product.Price,
			&product.ImageURL,
			&product.AverageRating,
			&product.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		products = append(products, &product)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetaData(totalRecords, filters.Page, filters.PageSize)
	return products, metadata, nil
}
