package image

import (
	"github.com/Semior001/mdcd-travelhack/app/utils"
	"github.com/go-pg/pg/v9"
	R "github.com/go-pkgz/rest"
	"github.com/pkg/errors"
	"log"
)

type PgImageStorage struct {
	db *pg.DB
}

func NewPgImageStorage(options pg.Options, logger *log.Logger) (*PgImageStorage, error) {
	db := pg.Connect(&options)
	pg.SetLogger(logger)
	return &PgImageStorage{
		db: db,
	}, nil
}

func (s *PgImageStorage) Migrate(force bool) error {
	log.Printf("[DEBUG] started image storage migration")
	if err := utils.CreateSchemas(s.db, force,
		(*Image)(nil),
	); err != nil {
		return errors.Wrapf(err, "there are some errors during the migration")
	}
	return nil
}

func (s *PgImageStorage) putImage(imgMetaData Image) (imgId uint64, err error) {
	if err := s.db.Insert(&imgMetaData); err != nil {
		return 0, err
	}
	return imgMetaData.ID, nil
}

func (s *PgImageStorage) getImage(id uint64) (Image, error) {
	image := Image{ID: id}
	if err := s.db.Select(&image); err != nil {
		return Image{}, err
	}
	return image, nil
}

func (s *PgImageStorage) GetBackgrounds() ([]uint64, error) {
	var ids []uint64
	var imgType string = "background"
	if err := s.db.Model((*Image)(nil)).Where("img_type = ?", imgType).ColumnExpr("array_agg(id)").Select(pg.Array(&ids)); err != nil {
		return nil, err
	}
	return ids, nil
}

func (s *PgImageStorage) CheckBarcode(barcode string) (json R.JSON, err error) {
	image := Image{BarCode: barcode}
	if err := s.db.Select(&image); err != nil {
		return R.JSON{}, err
	}
	return R.JSON{"ok": true}, nil
}

func (s *PgImageStorage) getImageByBarcode(barcode string) (imgMetaData *Image, err error) {
	image := Image{BarCode: barcode}
	if err := s.db.Model((*Image)(nil)).Where("bar_code = ?", barcode).Select(&image); err != nil {
		return nil, errors.Wrapf(err, "failed to load image from db by barcode")
	}
	return &image, nil
}
