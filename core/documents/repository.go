package documents

import (
	"context"
	"fmt"

	"github.com/Arup3201/torb/models"
	"gorm.io/gorm"
)

type DocumentRepository struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) *DocumentRepository {
	return &DocumentRepository{db}
}

func (r *DocumentRepository) Create(ctx context.Context,
	id, key, fileType, contentType string,
	size uint) error {

	doc := models.S3Document{
		ID:          id,
		Key:         key,
		ContentType: contentType,
		Type:        fileType,
		Size:        size,
	}
	err := gorm.G[models.S3Document](r.db).Create(ctx, &doc)
	if err != nil {
		return fmt.Errorf("gorm create: %w", err)
	}

	return nil
}
