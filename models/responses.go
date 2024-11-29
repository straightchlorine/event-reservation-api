package models

import (
	"time"

	"github.com/google/uuid"
)

// Standardized response for errors.
type ErrorResponse struct {
	Message string `json:"message" example:"Error message"`
}

// Standardized response for successful operations.
type SuccessResponse struct {
	Message string `json:"message" example:"Error message"`
}

// Standardized response for successful involving creating objects.
type SuccessResponseCreate struct {
	Message string `json:"message" example:"Error message"`
	ID      int    `json:"id"      example:"1"`
}

// Standardized response for successful involving creating objects.
type SuccessResponseCreateUUID struct {
	Message string `json:"message" example:"Error message"`
	UUID    string `json:"id"      example:"123e4567-e89b-12d3-a456-426614174000"`
}

// User ID and username.
type UserUsernameID struct {
	ID       string `json:"id"       example:"123e4567-e89b-12d3-a456-426614174000"`
	Username string `json:"username" example:"root"`
}

// Response after a successful login.
type LoginResponse struct {
	Token   string         `json:"token" example:"jwt-token-string"`
	Expires int64          `json:"exp"   example:"12313123"`
	User    UserUsernameID `json:"user"`
}

// Event, as its returned to user
type EventResponse struct {
	ID               int              `json:"id"                example:"1"`
	Name             string           `json:"name"              example:"Champions League Final"`
	Price            float64          `json:"price"             example:"99.99"`
	AvailableTickets int              `json:"available_tickets" example:"15000"`
	Date             time.Time        `json:"date"              example:"2024-12-31T20:00:00Z"`
	Location         LocationResponse `json:"location"`
}

// Location, as it is returned to the user.
type LocationResponse struct {
	ID       int    `json:"id"       example:"101"`
	Country  string `json:"country"  example:"England"`
	Address  string `json:"address"  example:"Wembley Park, London"`
	Stadium  string `json:"stadium"  example:"Wembley Stadium"`
	Capacity int    `json:"capacity" example:"90000"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Surname   string    `json:"surname"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	LastLogin time.Time `json:"last_login,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	RoleName  int       `json:"role_id"`
	IsActive  bool      `json:"is_active"`
}

type ReservationResponse struct {
	ID           string           `json:"id"`
	Username     string           `json:"user"`
	CreatedAt    time.Time        `json:"created_at"`
	TotalTickets int              `json:"total_tickets"`
	Status       string           `json:"status"`
	Event        EventResponse    `json:"event"`
	Tickets      []TicketResponse `json:"tickets"`
}