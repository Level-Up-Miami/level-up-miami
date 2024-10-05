package accountdatabase

import (
    "context"
    "fmt"
    "time"
    "github.com/jackc/pgx/v4/pgxpool"
    "golang.org/x/crypto/bcrypt"
)

// AccountDatabase represents the database for managing accounts
type AccountDatabase struct {
    Pool *pgxpool.Pool
}

// NewAccountDatabase initializes a new AccountDatabase instance
func NewAccountDatabase(dbURL string) (*AccountDatabase, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    pool, err := pgxpool.Connect(ctx, dbURL)
    if err != nil {
        return nil, fmt.Errorf("failed to connect to account database: %w", err)
    }

    // Initialize tables in the account database
    if err := InitializeAccountDatabase(pool); err != nil {
        return nil, fmt.Errorf("failed to initialize account database tables: %w", err)
    }

    return &AccountDatabase{Pool: pool}, nil
}

// InitializeAccountDatabase initializes the account-related tables
func InitializeAccountDatabase(pool *pgxpool.Pool) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    queries := []string{
        `
        CREATE TABLE IF NOT EXISTS accountsettings (
            account_id SERIAL PRIMARY KEY,
            username TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            password TEXT NOT NULL,
            email_verified BOOLEAN DEFAULT FALSE
        );
        `,
        `
        CREATE TABLE IF NOT EXISTS transaction_history (
            transaction_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            client_id TEXT NOT NULL,
            transaction_type TEXT NOT NULL,
            items_sent JSONB,
            items_received JSONB,
            notes TEXT,
            status TEXT DEFAULT 'Pending...'
        );
        `,
    }

    for _, q := range queries {
        if _, err := pool.Exec(ctx, q); err != nil {
            return fmt.Errorf("failed to execute query: %w", err)
        }
    }

    return nil
}

// CreateUser inserts a new user into the accountsettings table
func (db *AccountDatabase) CreateUser(username, password, email string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return fmt.Errorf("failed to hash password: %w", err)
    }

    query := `
        INSERT INTO accountsettings (username, password, email)
        VALUES ($1, $2, $3)
    `
    _, err = db.Pool.Exec(ctx, query, username, string(hashedPassword), email)
    if err != nil {
        return fmt.Errorf("failed to create user: %w", err)
    }

    return nil
}

// GetUserByUsername fetches a user by their username
func (db *AccountDatabase) GetUserByUsername(username string) (*User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var user User
    err := db.Pool.QueryRow(ctx, `
        SELECT account_id, username, email, password, email_verified
        FROM accountsettings
        WHERE username = $1
    `, username).Scan(&user.ID, &user.Username, &user.Email, &user.Password, &user.EmailVerified)
    if err != nil {
        return nil, fmt.Errorf("failed to get user by username: %w", err)
    }

    return &user, nil
}

// ValidateCredentials checks if the provided username and password are correct
func (db *AccountDatabase) ValidateCredentials(username, password string) (bool, bool, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var storedPassword string
    var emailVerified bool
    query := `SELECT password, email_verified FROM accountsettings WHERE username=$1`
    err := db.Pool.QueryRow(ctx, query, username).Scan(&storedPassword, &emailVerified)
    if err != nil {
        if err.Error() == "no rows in result set" {
            return false, false, nil // Username not found
        }
        return false, false, fmt.Errorf("failed to query accountsettings: %w", err)
    }

    // Check if the password matches
    if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
        return false, emailVerified, nil
    }

    return true, emailVerified, nil
}

// VerifyUserEmail sets the email_verified flag to true for a given username
func (db *AccountDatabase) VerifyUserEmail(username string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    query := `UPDATE accountsettings SET email_verified = TRUE WHERE username = $1`
    _, err := db.Pool.Exec(ctx, query, username)
    if err != nil {
        return fmt.Errorf("failed to verify email: %w", err)
    }

    return nil
}

// UpdateAccount updates account details in the accountsettings table
func (db *AccountDatabase) UpdateAccount(username, newUsername, newEmail string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    _, err := db.Pool.Exec(ctx, `
        UPDATE accountsettings
        SET username = COALESCE($1, username), email = COALESCE($2, email)
        WHERE username = $3
    `, newUsername, newEmail, username)

    if err != nil {
        return fmt.Errorf("failed to update account: %w", err)
    }

    return nil
}

// AddTransaction adds a new transaction to the transaction_history table
func (db *AccountDatabase) AddTransaction(clientID, transactionType, itemsSent, itemsReceived, notes string) error {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    query := `
        INSERT INTO transaction_history (client_id, transaction_type, items_sent, items_received, notes, status)
        VALUES ($1, $2, $3, $4, $5, 'Pending...')
    `
    _, err := db.Pool.Exec(ctx, query, clientID, transactionType, itemsSent, itemsReceived, notes)
    if err != nil {
        return fmt.Errorf("failed to add transaction: %w", err)
    }

    return nil
}

// User represents a user in the system
type User struct {
    ID            int
    Username      string
    Email         string
    Password      string
    EmailVerified bool
}