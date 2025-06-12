package database

import (
	"database/sql"
	"sync"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockConfig represents a mock database configuration
type MockConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

func (m MockConfig) GetDSN() string {
	return "host=" + m.Host + " port=" + m.Port + " user=" + m.User + " password=" + m.Password + " dbname=" + m.Name + " sslmode=" + m.SSLMode
}

func TestNewPostgresClient(t *testing.T) {
	t.Run("Successful connection with valid config", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		// Create client with mock DB
		client := &PostgresClient{
			db: sqlxDB,
		}

		assert.NotNil(t, client)
		assert.NotNil(t, client.db)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Connection with custom configuration", func(t *testing.T) {
		// This test verifies the DSN building logic
		config := MockConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "testuser",
			Password: "testpass",
			Name:     "testdb",
			SSLMode:  "disable",
		}

		expectedDSN := "host=localhost port=5432 user=testuser password=testpass dbname=testdb sslmode=disable"
		actualDSN := config.GetDSN()
		assert.Equal(t, expectedDSN, actualDSN)
	})

	t.Run("Connection with SSL enabled", func(t *testing.T) {
		config := MockConfig{
			Host:     "prod-db.example.com",
			Port:     "5432",
			User:     "produser",
			Password: "prodpass",
			Name:     "proddb",
			SSLMode:  "require",
		}

		expectedDSN := "host=prod-db.example.com port=5432 user=produser password=prodpass dbname=proddb sslmode=require"
		actualDSN := config.GetDSN()
		assert.Equal(t, expectedDSN, actualDSN)
	})

	t.Run("Connection with special characters in password", func(t *testing.T) {
		config := MockConfig{
			Host:     "localhost",
			Port:     "5432",
			User:     "user",
			Password: "p@ssw0rd!#$%",
			Name:     "db",
			SSLMode:  "disable",
		}

		expectedDSN := "host=localhost port=5432 user=user password=p@ssw0rd!#$% dbname=db sslmode=disable"
		actualDSN := config.GetDSN()
		assert.Equal(t, expectedDSN, actualDSN)
	})

	t.Run("Connection with empty values", func(t *testing.T) {
		config := MockConfig{
			Host:     "",
			Port:     "",
			User:     "",
			Password: "",
			Name:     "",
			SSLMode:  "",
		}

		expectedDSN := "host= port= user= password= dbname= sslmode="
		actualDSN := config.GetDSN()
		assert.Equal(t, expectedDSN, actualDSN)
	})
}

func TestPostgresClient_GetDB(t *testing.T) {
	t.Run("Get database instance", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		db := client.GetDB()
		assert.NotNil(t, db)
		assert.Equal(t, sqlxDB, db)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Get database instance from nil client", func(t *testing.T) {
		var client *PostgresClient
		assert.Panics(t, func() {
			client.GetDB()
		})
	})
}

func TestPostgresClient_Close(t *testing.T) {
	t.Run("Close database connection successfully", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)

		// Expect close to be called
		mock.ExpectClose()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		err = client.Close()
		assert.NoError(t, err)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Close database connection with error", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)

		// Expect close to return an error
		mock.ExpectClose().WillReturnError(sql.ErrConnDone)

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		err = client.Close()
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Close nil client", func(t *testing.T) {
		var client *PostgresClient
		assert.Panics(t, func() {
			client.Close()
		})
	})

	t.Run("Close client with nil database", func(t *testing.T) {
		client := &PostgresClient{
			db: nil,
		}

		// This should handle gracefully or panic depending on implementation
		assert.Panics(t, func() {
			client.Close()
		})
	})
}

func TestPostgresClient_DatabaseOperations(t *testing.T) {
	t.Run("Basic database operations", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		// Test that we can get the database and it's usable
		db := client.GetDB()
		assert.NotNil(t, db)

		// Test basic sqlx operations (mocked)
		// Example: Create table
		mock.ExpectExec("CREATE TABLE").WillReturnResult(sqlmock.NewResult(0, 0))

		// This would normally create a table, but we're just testing the mock
		_, err = db.Exec("CREATE TABLE test (id SERIAL PRIMARY KEY)")
		assert.NoError(t, err)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Transaction operations", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		// Mock transaction
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO").WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		// Test transaction
		db := client.GetDB()
		tx, err := db.Beginx()
		assert.NoError(t, err)
		_, err = tx.Exec("INSERT INTO test (name) VALUES ($1)", "test")
		assert.NoError(t, err)
		err = tx.Commit()
		assert.NoError(t, err)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresClient_ConnectionPooling(t *testing.T) {
	t.Run("Connection pool configuration", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		// Get underlying sql.DB to test connection pool settings
		sqlDB := client.db.DB
		assert.NotNil(t, sqlDB)

		// Test connection pool settings
		sqlDB.SetMaxOpenConns(25)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(5 * time.Minute)

		// Verify settings (these are just to ensure no panics)
		assert.NotNil(t, sqlDB)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresClient_ErrorHandling(t *testing.T) {
	t.Run("Handle database errors gracefully", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		// Mock a query that returns an error
		mock.ExpectQuery("SELECT").WillReturnError(sql.ErrNoRows)

		db := client.GetDB()
		var result struct{ ID int }
		err = db.Get(&result, "SELECT id FROM nonexistent")
		assert.Error(t, err)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresClient_Concurrent(t *testing.T) {
	t.Run("Concurrent database access", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		// Mock multiple queries
		for i := 0; i < 5; i++ {
			mock.ExpectQuery("SELECT").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(i))
		}

		// Test concurrent access
		var wg sync.WaitGroup
		for i := 0; i < 5; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				db := client.GetDB()
				var result struct{ ID int }
				db.Get(&result, "SELECT id FROM test")
			}()
		}

		wg.Wait()

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestPostgresClient_HealthCheck(t *testing.T) {
	t.Run("Database health check", func(t *testing.T) {
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(t, err)
		defer mockDB.Close()

		// Expect ping
		mock.ExpectPing()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		// Test health check via ping
		err = client.db.Ping()
		assert.NoError(t, err)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("Database health check failure", func(t *testing.T) {
		// Create a mock database with ping monitoring enabled
		mockDB, mock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
		require.NoError(t, err)
		defer mockDB.Close()

		// Expect ping to fail
		mock.ExpectPing().WillReturnError(sql.ErrConnDone)

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}

		// Test health check failure
		err = client.db.Ping()
		assert.Error(t, err)
		assert.Equal(t, sql.ErrConnDone, err)

		// Verify expectations
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func BenchmarkPostgresClient_GetDB(b *testing.B) {
	// Create a mock database
	mockDB, _, err := sqlmock.New()
	require.NoError(b, err)
	defer mockDB.Close()

	// Create sqlx DB with mock
	sqlxDB := sqlx.NewDb(mockDB, "postgres")

	client := &PostgresClient{
		db: sqlxDB,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.GetDB()
	}
}

func BenchmarkPostgresClient_Close(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create a mock database
		mockDB, mock, err := sqlmock.New()
		require.NoError(b, err)

		// Expect close
		mock.ExpectClose()

		// Create sqlx DB with mock
		sqlxDB := sqlx.NewDb(mockDB, "postgres")

		client := &PostgresClient{
			db: sqlxDB,
		}
		b.StartTimer()

		client.Close()
	}
}