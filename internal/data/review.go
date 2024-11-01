package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/martinezmoises/Test1/internal/validator"
)

type Review struct {
	ID           int64     `json:"id"`
	ProductID    int64     `json:"product_id"`
	Content      string    `json:"content"`
	Author       string    `json:"author"`
	Rating       int       `json:"rating"`
	HelpfulCount int       `json:"helpful_count"`
	CreatedAt    time.Time `json:"created_at"`
	Version      int32     `json:"version"`
}

type ReviewModel struct {
	DB *sql.DB
}

// ValidateReview checks the fields of a Review struct.
func ValidateReview(v *validator.Validator, review *Review) {
	v.Check(review.Content != "", "content", "must be provided")
	v.Check(len(review.Content) <= 500, "content", "must not exceed 500 characters")
	v.Check(review.Author != "", "author", "must be provided")
	v.Check(len(review.Author) <= 100, "author", "must not exceed 100 characters")
	v.Check(review.Rating >= 1 && review.Rating <= 5, "rating", "must be between 1 and 5")
}

// Insert creates a new review in the database.
func (m ReviewModel) Insert(review *Review) error {
	query := `
		INSERT INTO reviews (product_id, content, author, rating, helpful_count)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, created_at, version
	`
	args := []any{review.ProductID, review.Content, review.Author, review.Rating, review.HelpfulCount}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&review.ID, &review.CreatedAt, &review.Version)
}

// Get retrieves a specific review by its ID and associated product ID.
func (m ReviewModel) Get(productID, reviewID int64) (*Review, error) {
	query := `
		SELECT id, product_id, content, author, rating, helpful_count, created_at, version
		FROM reviews
		WHERE product_id = $1 AND id = $2
	`

	var review Review
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := m.DB.QueryRowContext(ctx, query, productID, reviewID).Scan(
		&review.ID,
		&review.ProductID,
		&review.Content,
		&review.Author,
		&review.Rating,
		&review.HelpfulCount,
		&review.CreatedAt,
		&review.Version,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, err
	}

	return &review, nil
}

// Update modifies an existing review.
func (m ReviewModel) Update(review *Review) error {
	query := `
		UPDATE reviews
		SET content = $1, author = $2, rating = $3, helpful_count = $4, version = version + 1
		WHERE product_id = $5 AND id = $6
		RETURNING version
	`
	args := []any{review.Content, review.Author, review.Rating, review.HelpfulCount, review.ProductID, review.ID}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return m.DB.QueryRowContext(ctx, query, args...).Scan(&review.Version)
}

// Delete removes a review by its ID and associated product ID.
func (m ReviewModel) Delete(productID, reviewID int64) error {
	query := `
		DELETE FROM reviews
		WHERE product_id = $1 AND id = $2
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	result, err := m.DB.ExecContext(ctx, query, productID, reviewID)
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

// GetAll retrieves all reviews for a specific product with filtering, sorting, and pagination.
func (m ReviewModel) GetAll(productID int64, filters Filters) ([]*Review, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(*) OVER(), id, product_id, content, author, rating, helpful_count, created_at, version
		FROM reviews
		WHERE (product_id = $1 OR $1 = 0)
		ORDER BY %s %s, id ASC
		LIMIT $2 OFFSET $3`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query, productID, filters.limit(), filters.offset())
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	reviews := []*Review{}

	for rows.Next() {
		var review Review
		err := rows.Scan(
			&totalRecords,
			&review.ID,
			&review.ProductID,
			&review.Content,
			&review.Author,
			&review.Rating,
			&review.HelpfulCount,
			&review.CreatedAt,
			&review.Version,
		)
		if err != nil {
			return nil, Metadata{}, err
		}
		reviews = append(reviews, &review)
	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetaData(totalRecords, filters.Page, filters.PageSize)
	return reviews, metadata, nil
}
