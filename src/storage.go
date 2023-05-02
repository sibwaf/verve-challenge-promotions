package src

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
)

// we're using two tables for atomical data switching:
// - one contains the live data (gets truncated after import)
// - one is for staging (becomes the primary one after import)

// a single table would be sufficient (we have table_id column to distinguish data versions),
// but cleaning up the old data via DELETE would require a full-scan which could be very slow

var DbClient *sql.DB
var writeMutex sync.Mutex

type Promotion struct {
	Id             uuid.UUID `json:"id"`
	Price          float64   `json:"price"`
	ExpirationDate time.Time `json:"expiration_date"`
}

func SavePromotions(promotions <-chan Promotion) error {
	writeMutex.Lock()
	defer writeMutex.Unlock()

	var currentTableId int
	err := DbClient.QueryRow("SELECT property_value FROM db_state WHERE property_name = 'current_table_id'").Scan(&currentTableId)
	if err != nil {
		return err
	}

	stagingTableId := (currentTableId + 1) % 2

	// we could cleanup the staging table after we've imported all data,
	// but if it fails, the next update will be operating over an incomplete dataset

	// we also have a way to switch to the old dataset on a faulty update if we keep the data

	_, err = DbClient.Exec(fmt.Sprintf("TRUNCATE promotions_%d", stagingTableId))
	if err != nil {
		return err
	}

	fileId, err := uuid.NewRandom()
	if err != nil {
		return err
	}

	// we could use batched inserts here, but bulk loads
	// are significantly faster for huge datasets

	mysql.RegisterReaderHandler(fileId.String(), func() io.Reader {
		return &promotionLoadConverter{
			Channel:   promotions,
			Separator: "\t",
		}
	})
	defer mysql.DeregisterReaderHandler(fileId.String())

	_, err = DbClient.Exec(
		fmt.Sprintf(
			"LOAD DATA LOCAL INFILE 'Reader::%s' "+
				"INTO TABLE promotions_%d "+
				"FIELDS TERMINATED BY '\t' "+
				"(id, price, expiration_date)",
			fileId.String(),
			stagingTableId,
		),
	)
	if err != nil {
		return err
	}

	_, err = DbClient.Exec(fmt.Sprintf("UPDATE db_state SET property_value = '%d' WHERE property_name = 'current_table_id'", stagingTableId))
	return err
}

type promotionLoadConverter struct {
	Channel   <-chan Promotion
	Separator string
	overflow  string
}

func (reader *promotionLoadConverter) Read(p []byte) (n int, err error) {
	if len(reader.overflow) > 0 {
		n = copy(p, reader.overflow)
		reader.overflow = reader.overflow[n:]
		return
	}

	promotion, ok := <-reader.Channel
	if !ok {
		err = io.EOF
		return
	}

	row := strings.Join(
		[]string{
			promotion.Id.String(),
			fmt.Sprintf("%f", promotion.Price),
			promotion.ExpirationDate.Format(time.DateTime),
		},
		reader.Separator,
	) + "\n"

	n = copy(p, row)
	reader.overflow = row[n:]
	return
}

func FindPromotionById(id uuid.UUID) (Promotion, error) {
	var price float64
	var expirationDate time.Time

	currentTableIdQuery := "SELECT property_value FROM db_state WHERE property_name = 'current_table_id'"

	err := DbClient.QueryRow(
		"SELECT price, expiration_date "+
			"FROM promotions_0 WHERE id = ? AND table_id = ("+currentTableIdQuery+")"+
			"UNION ALL "+
			"SELECT price, expiration_date "+
			"FROM promotions_1 WHERE id = ? AND table_id = ("+currentTableIdQuery+")",
		id.String(),
		id.String(),
	).Scan(&price, &expirationDate)

	if err != nil {
		return Promotion{}, err
	}

	return Promotion{
		Id:             id,
		Price:          price,
		ExpirationDate: expirationDate,
	}, nil
}
