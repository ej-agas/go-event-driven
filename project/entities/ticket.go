package entities

type Ticket struct {
	ID            string `json:"ticket_id"`
	Status        string `json:"status"`
	Price         Price  `json:"price"`
	CustomerEmail string `json:"customer_email"`
}
