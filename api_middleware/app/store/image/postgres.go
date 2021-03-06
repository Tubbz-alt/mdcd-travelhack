package image

import (
	"github.com/Semior001/mdcd-travelhack/app/store/user"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"log"
	"time"
)

// PgStore is a store interface implementation over PostgresSQL
type PgStore struct {
	ConnStr string

	connPool *pgx.ConnPool
}

// NewPgStore creates a connection pool to the postgres storage
func NewPgStore(connStr string) (*PgStore, error) {
	connConf, err := pgx.ParseConnectionString(connStr)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse pg image store with connstr %s", connStr)
	}

	p, err := pgx.NewConnPool(pgx.ConnPoolConfig{
		ConnConfig:     connConf,
		MaxConnections: 5,
		AfterConnect: func(conn *pgx.Conn) error {
			// todo no-op yet
			return nil
		},
		AcquireTimeout: time.Minute,
	})

	if err != nil {
		return nil, errors.Wrapf(err, "failed to initialize pg image store with connstr %s", connStr)
	}

	return &PgStore{
		ConnStr:  connStr,
		connPool: p,
	}, nil
}

// putImage stores image in database and returns its id
func (p *PgStore) putImage(image Image) (uint64, error) {
	tx, err := p.connPool.Begin()
	if err != nil {
		return 0, errors.Wrap(err, "failed to start insert transaction into pg image store")
	}

	defer func() {
		errNested := tx.Rollback()
		if errNested != nil && errNested != pgx.ErrTxClosed {
			log.Printf("[ERROR] failed to rollback the transaction (putImage): %+v", err)
		}
	}()

	row := p.connPool.QueryRow("INSERT INTO "+
		"images(bar_code, mime, img_type, local_filename, user_id, created_at, updated_at) "+
		"VALUES($1, $2, $3, $4, $5, $6, $7) "+
		"RETURNING id",
		image.Barcode, image.Mime, image.ImgType, image.LocalFilename, image.UserID,
		time.Now(), time.Now())

	var id uint64
	err = row.Scan(&id)
	if err != nil {
		return 0, errors.Wrap(err, "failed to scan image ID while inserting")
	}

	err = tx.Commit()

	if err != nil {
		return 0, errors.Wrap(err, "failed to commit transaction putImage")
	}

	return id, nil
}

// getImage returns image by its id
func (p *PgStore) getImage(id uint64) (Image, error) {
	image := Image{ID: id}
	u := user.User{}

	row := p.connPool.QueryRow("SELECT i.bar_code, i.mime, "+
		"i.img_type, i.local_filename, i.user_id, i.created_at, i.updated_at, "+
		"u.email, u.password, u.privileges, u.created_at, u.updated_at "+
		"FROM images AS i "+
		"INNER JOIN users AS u "+
		"ON i.user_id = u.id "+
		"WHERE i.id = $1", id)

	err := row.Scan(&image.Barcode, &image.Mime, &image.ImgType,
		&image.LocalFilename, &image.UserID, &image.CreatedAt, &image.UpdatedAt,
		&u.Email, &u.Password, &u.Privileges, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return Image{}, errors.Wrapf(err, "failed to scan image with id = %d", id)
	}

	u.ID = image.UserID
	image.AddedBy = &u

	return image, nil
}

// GetBackgroundIds returns slice with ids of all background images
func (p *PgStore) GetBackgroundIds() ([]uint64, error) {
	var ids []uint64

	rows, err := p.connPool.Query("SELECT id FROM images WHERE img_type = $1", ImgTypeBackground)
	if err != nil {
		return nil, errors.Wrap(err, "failed to select all background ids")
	}

	defer rows.Close()

	for rows.Next() {
		var id uint64

		err = rows.Scan(&id)
		if err != nil {
			return nil, errors.Wrap(err, "failed to scan one of images")
		}

		ids = append(ids, id)
	}

	if rows.Err() != nil {
		return nil, errors.Wrap(rows.Err(), "failed to process all rows in GetBackgroundIds method")
	}

	return ids, nil
}

// CheckBarcode returns true if barcode is present in database
func (p *PgStore) CheckBarcode(barcode string) (bool, error) {
	row := p.connPool.QueryRow("SELECT id FROM images WHERE bar_code = $1", barcode)

	err := row.Scan(nil)
	if err == pgx.ErrNoRows {
		return false, nil
	}

	if err != nil {
		return false, errors.Wrapf(err, "Failed to scan image (barcode = %s) in CheckBarcode method",
			barcode)
	}

	return true, nil
}

// getImgByBarcode returns image by its barcode
func (p *PgStore) getImgByBarcode(barcode string) (Image, error) {
	image := Image{Barcode: barcode}
	u := user.User{}

	row := p.connPool.QueryRow("SELECT i.id, i.mime, i.img_type, "+
		"i.local_filename, i.user_id, i.created_at, i.updated_at, "+
		"u.email, u.password, u.privileges, u.created_at, u.updated_at "+
		"FROM images AS i "+
		"INNER JOIN users AS u "+
		"ON i.user_id = u.id "+
		"WHERE i.bar_code = $1", barcode)

	err := row.Scan(&image.ID, &image.Mime, &image.ImgType,
		&image.LocalFilename, &image.UserID, &image.CreatedAt, &image.UpdatedAt,
		&u.Email, &u.Password, &u.Privileges, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return Image{}, errors.Wrapf(err, "failed to scan image with barcode = %s", barcode)
	}

	u.ID = image.UserID
	image.AddedBy = &u

	return image, nil
}
