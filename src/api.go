package src

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

func GetApiRoutes() map[string]http.HandlerFunc {
	routes := make(map[string]http.HandlerFunc)
	routes["/promotions/"] = getPromotion
	return routes
}

func getPromotion(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/promotions/")
	parsedId, err := uuid.Parse(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	promotion, err := FindPromotionById(parsedId)
	if err != nil {
		if err == sql.ErrNoRows {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println("Failed to handle getPromotion", err)
		}
		return
	}

	jsonBytes, err := promotion.marshalJson()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(jsonBytes)
}

func (promotion Promotion) marshalJson() ([]byte, error) {
	return json.Marshal(&struct {
		Id             string  `json:"id"`
		Price          float64 `json:"price"`
		ExpirationDate string  `json:"expiration_date"`
	}{
		Id:             promotion.Id.String(),
		Price:          promotion.Price,
		ExpirationDate: promotion.ExpirationDate.Format(time.DateTime),
	})
}
