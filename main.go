package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"io/ioutil"
	"log"
	"net/http"
)

type user struct{
	Id int `json:"id"`
	Name string `json:"name"`
	Money sql.NullInt64 `json:"money"`
}
var users []user
var db *sql.DB
var err error

type JsUSD struct {
	Rates struct{
		USD float64 `json:"USD"`
		RUB float64 `json:"RUB"`
	}
}



func GetBalance(w http.ResponseWriter, r *http.Request){

	w.Header().Set("Content-Type", "application/json")
	param := mux.Vars(r)
	rows, err := db.Query("SELECT money FROM users WHERE user_id = $1",param["id"])
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next(){
		u := user{}
		err := rows.Scan(&u.Money)
		if err != nil{
			fmt.Println(err)
			continue
		}
		json.NewEncoder(w).Encode(u.Money.Int64)
	}

}

func GetBalanceUSD(w http.ResponseWriter, r *http.Request){

	w.Header().Set("Content-Type", "application/json")
	param := mux.Vars(r)
	rows, err := db.Query("SELECT money FROM users WHERE user_id = $1",param["id"])
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	jsusd :=JsUSD{}
	ApiURL := fmt.Sprintf("http://api.exchangeratesapi.io/v1/latest?access_key=592c45704657af450de5f9c1034f5c91")
	resp , err := http.Get(ApiURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(body, &jsusd)
	if err != nil {
		log.Fatal(err)
	}

	OneDollar := jsusd.Rates.RUB/jsusd.Rates.USD

	for rows.Next(){
		u := user{}
		err := rows.Scan(&u.Money)
		if err != nil{
			fmt.Println(err)
			continue
		}
		json.NewEncoder(w).Encode(float64(u.Money.Int64)/(OneDollar))
	}
}

func ReplenishmentBalance(w http.ResponseWriter, r *http.Request){

	w.Header().Set("Content-Type", "application/json")
	param := mux.Vars(r)
	var us user
	_ = json.NewDecoder(r.Body).Decode(&us)
	result, err := db.Exec("UPDATE users SET money = money + $1 where user_id = $2",us.Money.Int64,param["id"])
	if err != nil{
		panic(err)
	}
	json.NewEncoder(w).Encode(us)
	PrintUpdate, _ := result.RowsAffected()
	fmt.Println(PrintUpdate,us.Money.Int64)
}

func Debit(w http.ResponseWriter, r *http.Request){

	w.Header().Set("Content-Type", "application/json")
	param := mux.Vars(r)
	var us user
	_ = json.NewDecoder(r.Body).Decode(&us)
	fmt.Println(us)

	rows, err := db.Query("SELECT money FROM users WHERE user_id = $1",param["id"])
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next(){
		u := user{}
		err := rows.Scan(&u.Money)
		if err != nil{
			fmt.Println(err)
			continue
		}
		if us.Money.Int64 > u.Money.Int64{
			error := "Недостаточно средств"
			json.NewEncoder(w).Encode(error)
			} else {
			result, err := db.Exec("UPDATE users SET money = money - $1 where user_id = $2",us.Money.Int64,param["id"])
			if err != nil{
				panic(err)
			}
			json.NewEncoder(w).Encode(us)
			PrintUpdate, _ := result.RowsAffected()
			fmt.Println(PrintUpdate,us.Money.Int64)
		}
	}




}

func Swap(w http.ResponseWriter, r *http.Request){

	w.Header().Set("Content-Type", "application/json")
	param := mux.Vars(r)
	fmt.Println(param["id"])
	var us user
	_ = json.NewDecoder(r.Body).Decode(&us)
	rows, err := db.Query("SELECT money FROM users WHERE user_id = $1",param["id"])
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next(){
		u := user{}
		err := rows.Scan(&u.Money)
		if err != nil{
			fmt.Println(err)
			continue
		}
		if us.Money.Int64 > u.Money.Int64{
			error := "Недостаточно средств"
			json.NewEncoder(w).Encode(error)
		} else {
			result, err := db.Exec("UPDATE users SET money = money + $1 where user_id = $2; ",us.Money.Int64,us.Id)
			if err != nil{
				panic(err)
			}
			_, err = db.Exec("UPDATE users SET money = money - $1 where user_id = $2; ", us.Money.Int64, param["id"])
			if err != nil{
				panic(err)
			}
			json.NewEncoder(w).Encode(us)
			PrintUpdate, _ := result.RowsAffected()
			fmt.Println(PrintUpdate,us.Money.Int64)
		}
	}
}

func main() {

	pg_con_string := fmt.Sprintf("port=%d host=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		5432, "localhost", "habrpguser", "passwd", "habrdb")

	db, err = sql.Open("postgres", pg_con_string)
	if err != nil {
		panic(err)
	}
	err = db.Ping()
	defer db.Close()


	r := mux.NewRouter()
	r.HandleFunc("/users/{id}",GetBalance).Methods("GET")
	r.HandleFunc("/users/{id}/usd",GetBalanceUSD).Methods("GET")
	r.HandleFunc("/users/repl/{id}",ReplenishmentBalance).Methods("PUT")
	r.HandleFunc("/users/deb/{id}",Debit).Methods("PUT")
	r.HandleFunc("/users/swap/{id}",Swap).Methods("PUT")

	log.Fatal(http.ListenAndServe(":8080",r))

}