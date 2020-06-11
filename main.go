package main

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"net/http"
	"text/template"
	"time"
)

var client *mongo.Client

type Payment struct {
	ID          primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	Value       float32            `json:"value,omitempty" bson:"value,omitempty"`
	Description string             `json:"description,omitempty" bson:"description,omitempty"`
}

type ResponseURL struct {
	Url string `json:"url,omitempty" bson:"url,omitempty"`
}

func AddPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	var payment Payment
	decodeErr := json.NewDecoder(r.Body).Decode(&payment)
	if decodeErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "` + decodeErr.Error() + `"}`))
		return
	}
	collection := client.Database("billing").Collection("payments")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, insertErr := collection.InsertOne(ctx, payment)
	if insertErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "` + insertErr.Error() + `"}`))
		return
	}
	var response ResponseURL
	url := "http://localhost:12345/payments/card/form?sessionId=" + result.InsertedID.(primitive.ObjectID).Hex()
	response.Url = url
	encodeErr := json.NewEncoder(w).Encode(response)
	if encodeErr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "` + encodeErr.Error() + `"}`))
		return
	}
}

type tplParams struct {
	Value       float32
	Description string
}

const formTmpl = `
<html>
	<body>
		Payment Value: {{.Value}}
		Description: {{.Description}}
		<form action="/luhn" method="post">
			Card Number: <input type="text" name="cardNumber">
			<input type="submit" value="Pay">
		</form>
	</body>
</html>
`

func ProcessPayment(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	tmpl := template.New(`cardForm`)
	tmpl, _ = tmpl.Parse(formTmpl)
	sessId := r.URL.Query().Get("sessionId")
	id, _ := primitive.ObjectIDFromHex(sessId)
	var payment Payment
	collection := client.Database("billing").Collection("payments")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err := collection.FindOne(ctx, Payment{ID: id}).Decode(&payment)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "` + err.Error() + `"}`))
		return
	}
	params := tplParams{
		Value:       payment.Value,
		Description: payment.Description,
	}
	tmpl.Execute(w, params)
}

func IsValid(s string) bool {
	sum := 0
	n := len(s)
	parity := (n - 1) % 2
	for i := n; i > 0; i-- {
		c := int(s[i-1]) - int('0')
		if parity == i%2 {
			c *= 2
		}
		sum += c / 10
		sum += c % 10
	}
	return sum%10 == 0
}

func CheckLuhn(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "text/html")
	cardNumber := r.FormValue("cardNumber")
	if IsValid(cardNumber) {
		w.Write([]byte("Payment is successful!"))
	} else {
		w.Write([]byte("Payment is unsuccessful"))
	}
}

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, _ = mongo.Connect(ctx, clientOptions)
	router := mux.NewRouter()
	router.HandleFunc("/register", AddPayment).Methods("POST")
	router.HandleFunc("/payments/card/form", ProcessPayment).Methods("GET")
	router.HandleFunc("/luhn", CheckLuhn).Methods("POST")
	http.ListenAndServe(":12345", router)
}
