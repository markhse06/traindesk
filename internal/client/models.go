package client

// CreateClientRequest — тело запроса для создания клиента.
type CreateClientRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// TODO: при необходимости добавить phone/email/notes.
}

// ClientResponse — то, что возвращаем клиенту во всех клиентских эндпоинтах.
type ClientResponse struct {
	ID        string `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	// TODO: добавить дату рождения и тд.
}
