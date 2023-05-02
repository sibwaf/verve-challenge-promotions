package src

import (
	"encoding/csv"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func GetUpdaterRoutes() map[string]http.HandlerFunc {
	routes := make(map[string]http.HandlerFunc)
	routes["/upload"] = upload
	return routes
}

func upload(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("data")
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	reader := csv.NewReader(file)
	reader.ReuseRecord = true
	reader.FieldsPerRecord = 3

	promotions := make(chan Promotion)
	defer close(promotions)

	go SavePromotions(promotions)

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		id, err := uuid.Parse(record[0])
		if err != nil {
			continue
		}

		price, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}

		expirationDate, err := time.Parse("2006-01-02 15:04:05 -0700 MST", record[2])
		if err != nil {
			continue
		}

		promotions <- Promotion{
			Id:             id,
			Price:          price,
			ExpirationDate: expirationDate.UTC(),
		}
	}
}
