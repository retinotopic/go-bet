package queue

type Message struct {
	User_id  int `json:"user_id"`
	Rating   int `json:"rating"`
	Attempts int `json:"attempts"`
}

type Config struct {
	Consume  Consume  `json:"consume"`
	Exchange Exchange `json:"exchange"`
}
