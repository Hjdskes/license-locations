package main

import (
    "fmt"
    "log"
    "time"
    "database/sql"

    _ "github.com/go-sql-driver/mysql"
    "github.com/google/go-github/github"
    "golang.org/x/oauth2"
)

type User struct {
  login string             // The user's login name.
  city string              // The user's reported city.
  state string             // The user's reported state or province.
  country string           // The user's reported country.
  licenses map[string]int  // A hashmap from license name to a count of how many
                           // times this user has used that license.
}

// An array holding all licenses currently tracked by GitHub. Used as keys into
// the `User.licenses` hashmap and as columns into the database.
var gh_licenses = []string{"license_other", "license_wtfpl", "license_lgpl30",
                           "license_bsd3", "license_unlicense", "license_lgpl21",
                           "license_apache20", "license_bsd2", "license_epl10",
                           "license_agpl30", "license_mit", "license_gpl20",
                           "license_mpl20", "license_gpl30"}

// updateDatabase loops over the licenses the passed user uses and updates the
// counts kept in the database.
func updateDatabase(user *User, tx *sql.Tx) {
    for license, count := range user.licenses {
        if count > 0 {
            // I know this is ugly, but the data isn't user generated so it
            // should be fine for our purposes.
            query := fmt.Sprintf("UPDATE locations SET %s=%s+%d, developers=developers+1 WHERE city=\"%s\" AND state=\"%s\" AND country=\"%s\"",
                                 license, license, count, user.city, user.state, user.country)
            _, err := tx.Exec(query)
            if err != nil {
                log.Println(query)
          	    log.Println(err)
	            tx.Rollback()
	            return
            }
        }
    }

    tx.Commit()
}

// countLicensesForUser attempts to retrieve the license of each original
// (non-forked) repository owned by a user. When the rate limit is hit, this
// will sleep until the we can make more requests again.
func countLicensesForUser(user *User, client *github.Client, opt *github.RepositoryListOptions) {
    repos, _, err := client.Repositories.List(user.login, opt)

    if rateLimitErr, ok := err.(*github.RateLimitError); ok {
        duration := rateLimitErr.Rate.Reset.Time.Sub(time.Now()) + 5 * time.Minute
	    log.Println("Rate limit met. Sleeping for ", duration)
	    time.Sleep(duration)
    }

    for _, repo := range repos {
        if !*repo.Fork && repo.License != nil {
            switch *repo.License.Key {
                // Copyleft
                case "gpl-2.0": user.licenses["license_gpl20"] += 1
                case "gpl-3.0": user.licenses["license_gpl30"] += 1
                case "lgpl-2.1": user.licenses["license_lgpl21"] += 1
                case "lgpl-3.0": user.licenses["license_lgpl30"] += 1
                case "agpl-3.0": user.licenses["license_agpl30"] += 1
                // Weak copyleft
                case "mpl-2.0": user.licenses["license_mpl20"] += 1
                case "epl-1.0": user.licenses["license_epl10"] += 1
                // Non-copyleft
                case "mit": user.licenses["license_mit"] += 1
                case "bsd-3-clause": user.licenses["license_bsd3"] += 1
                case "bsd-2-clause": user.licenses["license_bsd2"] += 1
                case "apache-2.0": user.licenses["license_apache20"] += 1
                case "unlicense": user.licenses["license_unlicense"] += 1
                case "wtfpl": user.licenses["license_wtfpl"] += 1
                // Other
                default: user.licenses["license_other"] += 1
            }
        }
    }
}

// setupGitHub connects to the GitHub API using the provided OAuth2 token.
func setupGitHub(token string) *github.Client {
    tokenSource := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tokenClient := oauth2.NewClient(oauth2.NoContext, tokenSource)
    return github.NewClient(tokenClient)
}

// setupDatabase connects to the database using the provided driver and source.
func setupDatabase(driverName, dataSourceName string) *sql.DB {
    db, err := sql.Open(driverName, dataSourceName)
    if err != nil {
        log.Fatal(err)
    }
    return db
}

func main() {
    // Connect to the database.
    db := setupDatabase("mysql", "ghtorrentuser:ghtorrentpassword@tcp(127.0.0.1:3306)/ghtorrent_restore")
    defer db.Close()
    if err := db.Ping(); err != nil {
        log.Fatal(err)
    }

    // Connect to GitHub.
    client := setupGitHub(API_KEY_PLACEHOLDER)

    // Get the first five hundred users from the `users` table.
    rows, err := db.Query(`SELECT login, city, state, country_code FROM users
                           WHERE location IS NOT NULL
                             AND country_code IS NOT NULL
                             AND state IS NOT NULL
                             AND city IS NOT NULL
                             AND deleted=0 AND fake=0 AND type='USR'
                           LIMIT 5000`)
    if err != nil {
        log.Fatal(err)
    }
    defer rows.Close()

    // For each user, request a count of all the licenses they use and store
    // this information in the `locations` table.
    opt := &github.RepositoryListOptions{Type: "owner"}
    for rows.Next() {
        var user User
        user.licenses = make(map[string]int)
        err := rows.Scan(&user.login, &user.city, &user.state, &user.country)
        if err != nil {
            log.Fatal(err)
        }

        countLicensesForUser(&user, client, opt)
        tx, err := db.Begin()
        if err != nil {
            log.Print("Skipping count for user %s: ", user.login)
            log.Println(err)
        } else {
            updateDatabase(&user, tx)
        }
    }
    if err = rows.Err(); err != nil {
        log.Fatal(err)
    }
}

