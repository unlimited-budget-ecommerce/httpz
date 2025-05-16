package httpz

import (
	gojson "encoding/json"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

type smallBody struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
}

func TestJsonAndGoJsonMarshal(t *testing.T) {
	body := smallBody{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@email.com",
		Age:       30,
	}

	b1, err := json.Marshal(body)

	assert.NoError(t, err)

	b2, err := gojson.Marshal(body)

	assert.NoError(t, err)
	assert.Equal(t, b1, b2)
}

func BenchmarkSmallJsonMarshal(b *testing.B) {
	body := smallBody{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@email.com",
		Age:       30,
	}

	for b.Loop() {
		_, _ = json.Marshal(body)
	}
}

func BenchmarkSmallGoJsonMarshal(b *testing.B) {
	body := smallBody{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@email.com",
		Age:       30,
	}

	for b.Loop() {
		_, _ = gojson.Marshal(body)
	}
}

func BenchmarkSmallJsonUnmarshal(b *testing.B) {
	res := `{"first_name":"John","last_name":"Doe","email":"john.doe@email.com","age":30}`

	for b.Loop() {
		body := smallBody{}
		_ = json.Unmarshal([]byte(res), &body)
	}
}

func BenchmarkSmallGoJsonUnmarshal(b *testing.B) {
	res := `{"first_name":"John","last_name":"Doe","email":"john.doe@email.com","age":30}`

	for b.Loop() {
		body := smallBody{}
		_ = gojson.Unmarshal([]byte(res), &body)
	}
}

type mediumBody struct {
	FirstName string   `json:"first_name"`
	LastName  string   `json:"last_name"`
	Email     string   `json:"email"`
	Age       int      `json:"age"`
	MobileNo  string   `json:"mobile_no"`
	Height    float64  `json:"height"`
	Weight    float64  `json:"weight"`
	Address   address  `json:"address"`
	Hobbies   []string `json:"hobbies"`
	Interests []string `json:"interests"`
	IsMarried bool     `json:"is_married"`
}

type address struct {
	AddressNo   string `json:"address_no"`
	VillageNo   string `json:"village_no"`
	Alley       string `json:"alley"`
	Street      string `json:"street"`
	District    string `json:"district"`
	SubDistrict string `json:"sub_district"`
	Province    string `json:"province"`
	Country     string `json:"country"`
	ZipCode     string `json:"zip_code"`
	PhoneNo     string `json:"phone_no"`
}

func BenchmarkMediumJsonMarshal(b *testing.B) {
	body := mediumBody{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@email.com",
		Age:       30,
		MobileNo:  "0123456789",
		Height:    179.8,
		Weight:    70.5,
		Address: address{
			AddressNo:   "123",
			VillageNo:   "456",
			Alley:       "789",
			Street:      "Main Street",
			District:    "District",
			SubDistrict: "Sub District",
			Province:    "Province",
			Country:     "Country",
			ZipCode:     "12345",
			PhoneNo:     "0123456789",
		},
		Hobbies:   []string{"abc1234", "def1234", "ghi1234"},
		Interests: []string{"jkl1234", "mno1234", "pqr1234"},
		IsMarried: true,
	}

	for b.Loop() {
		_, _ = json.Marshal(body)
	}
}

func BenchmarkMediumGoJsonMarshal(b *testing.B) {
	body := mediumBody{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john.doe@email.com",
		Age:       30,
		MobileNo:  "0123456789",
		Height:    179.8,
		Weight:    70.5,
		Address: address{
			AddressNo:   "123",
			VillageNo:   "456",
			Alley:       "789",
			Street:      "Main Street",
			District:    "District",
			SubDistrict: "Sub District",
			Province:    "Province",
			Country:     "Country",
			ZipCode:     "12345",
			PhoneNo:     "0123456789",
		},
		Hobbies:   []string{"abc1234", "def1234", "ghi1234"},
		Interests: []string{"jkl1234", "mno1234", "pqr1234"},
		IsMarried: true,
	}

	for b.Loop() {
		_, _ = gojson.Marshal(body)
	}
}

func BenchmarkMediumJsonUnmarshal(b *testing.B) {
	res := `{"first_name":"John","last_name":"Doe","email":"john.doe@email.com","age":30,"mobile_no":"0123456789","height":179.8,"weight":70.5,"address":{"address_no":"123","village_no":"456","alley":"789","street":"Main Street","district":"District","sub_district":"Sub District","province":"Province","country":"Country","zip_code":"12345","phone_no":"0123456789"},"hobbies":["abc1234","def1234","ghi1234"],"interests":["jkl1234","mno1234","pqr1234"],"is_married":true}`

	for b.Loop() {
		body := mediumBody{}
		_ = json.Unmarshal([]byte(res), &body)
	}
}

func BenchmarkMediumGoJsonUnmarshal(b *testing.B) {
	res := `{"first_name":"John","last_name":"Doe","email":"john.doe@email.com","age":30,"mobile_no":"0123456789","height":179.8,"weight":70.5,"address":{"address_no":"123","village_no":"456","alley":"789","street":"Main Street","district":"District","sub_district":"Sub District","province":"Province","country":"Country","zip_code":"12345","phone_no":"0123456789"},"hobbies":["abc1234","def1234","ghi1234"],"interests":["jkl1234","mno1234","pqr1234"],"is_married":true}`

	for b.Loop() {
		body := mediumBody{}
		_ = gojson.Unmarshal([]byte(res), &body)
	}
}
