package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// Twitter represents a twitter in the Hacker News clone
type Twitter struct {
	ID        int
	Content   string
	CreatedAt time.Time
}

// createTable encapsulates the logic to create a table.
// It checks if the table exists in the 'public' schema, and creates it if not.
func createTable(db *sql.DB, tableName, createQuery string) error {
	var exists bool
	// SQL query to check if the table exists
	err := db.QueryRow(`
        SELECT EXISTS (
            SELECT FROM information_schema.tables 
            WHERE table_schema = 'public' 
            AND table_name = $1
        );
    `, tableName).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		// Create the table if it doesn't exist
		_, err := db.Exec(createQuery)
		if err != nil {
			return err
		}
		fmt.Printf("%s table created.\n", tableName)
	}
	return nil
}

// createTables creates all necessary tables.
func createTables(db *sql.DB) error {
	// SQL query to create the 'twitters' table
	twittersTableQuery := `
        CREATE TABLE twitters (
            id SERIAL PRIMARY KEY, -- Auto - incrementing primary key
            content TEXT NOT NULL, -- Content of the twitter
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP -- Creation time
        );
    `
	if err := createTable(db, "twitters", twittersTableQuery); err != nil {
		return err
	}

	return nil
}

// renderTemplate encapsulates the template rendering logic.
func renderTemplate(c *gin.Context, tmplPath string, data interface{}) {
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(c.Writer, data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}

func main() {

	// Get the DSN from environment variable
	dsn := os.Getenv("DSN")

	// Connect to the database using DSN
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Create tables if they don't exist
	if err := createTables(db); err != nil {
		log.Fatal(err)
	}

	// Set up Gin router
	r := gin.Default()

	// Serve static files
	r.Static("/static", "./static")

	// Define routes
	// Route to display the list of twitters
	r.GET("/", func(c *gin.Context) {
		// SQL query to select all twitters ordered by creation time in descending order
		rows, err := db.Query("SELECT id, content, created_at FROM twitters ORDER BY created_at DESC")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer rows.Close()

		var twitters []Twitter
		for rows.Next() {
			var twitter Twitter
			if err := rows.Scan(
				&twitter.ID,
				&twitter.Content,
				&twitter.CreatedAt,
			); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}

			twitters = append(twitters, twitter)
		}
		if err := rows.Err(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		renderTemplate(c, "templates/index.html", map[string]interface{}{
			"Twitters": twitters,
		})
	})

	// Route to add a new twitter
	r.POST("/new", func(c *gin.Context) {
		content := c.PostForm("content")
		// SQL query to insert a new twitter into the 'twitters' table
		if _, err := db.Exec("INSERT INTO twitters (content, created_at) VALUES ($1, CURRENT_TIMESTAMP)", content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Redirect(http.StatusFound, "/")
	})

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server started on port %s", port)
	r.Run(":" + port)
}
