# License locations

This repository holds the proof-of-concept code for the draft research paper I
wrote on software licenses for the course IN4252 Web Science & Engineering at
the Delft University of Technology. The paper is titled *Open source software
licenses: is there a correlation between developer location and software
licenses?*. The data for the research comes from the
[GHTorrent](http://ghtorrent.org/) project, but had to be augmented with license
data from GitHub because GHTorrent does not
([yet?](https://github.com/gousiosg/github-mirror/issues/38)) track license
information.

This repository holds the code to do so. Since I was writing a draft research 
paper, I only queried GitHub for the first 5000 users to verify that the setup
works. The database dump I used from GHTorrent had 740000 users; querying all
those users with GitHub's 5000 requests per hour rate limit would simply take
too long. This means that the code can probably be improved to improve speed,
increase robustness and ease of use. It is also my first real project using
Golang. Pull requests are accepted!

## Setup

First, you need a datadump from GHTorrent. I used the 2016-12-01 dump. You only
need to import `users.csv` and `projects.csv`; the other data is not required.
See the README in your downloaded datadump for how to restore it.

Next, create a table called `locations` using the following SQL statement:

```sql
CREATE TABLE IF NOT EXISTS locations (
  developers int NOT NULL DEFAULT 0,
  city VARCHAR(255) NOT NULL,
  state VARCHAR(255) NOT NULL,
  country VARCHAR(255) NOT NULL,
  license_other int NOT NULL DEFAULT 0,
  license_wtfpl int NOT NULL DEFAULT 0,
  license_lgpl30 int NOT NULL DEFAULT 0,
  license_bsd3 int NOT NULL DEFAULT 0,
  license_unlicense int NOT NULL DEFAULT 0,
  license_lgpl21 int NOT NULL DEFAULT 0,
  license_apache20 int NOT NULL DEFAULT 0,
  license_bsd2 int NOT NULL DEFAULT 0,
  license_epl10 int NOT NULL DEFAULT 0,
  license_agpl30 int NOT NULL DEFAULT 0,
  license_mit int NOT NULL DEFAULT 0,
  license_gpl20 int NOT NULL DEFAULT 0,
  license_mpl20 int NOT NULL DEFAULT 0,
  license_gpl30 int NOT NULL DEFAULT 0,
  PRIMARY KEY (city, state, country)
)
```

Finally, populate this table with all `city, state, country` combinations found
in the `users` table:

```sql
INSERT INTO locations (city,state,country)
SELECT users.city, users.state, users.country_code
FROM users
WHERE users.location IS NOT NULL
  AND (users.country_code IS NOT NULL AND users.state IS NOT NULL AND users.city IS NOT NULL)
  AND users.deleted=0
  AND users.fake=0
  AND users.type='USR'
ON DUPLICATE KEY UPDATE city=users.city, state=users.state, country=users.country_code
```

Before you can run the Go program, you need to obtain
[an access token](https://help.github.com/articles/creating-an-access-token-for-command-line-use/)
which you will need to copy over `API_KEY_PLACEHOLDER` in the call to
`setupGitHub`. You can now run the Go program to query GitHub for licenses used
by the first 5000 developers from the `users` table.

## Queries used to analyze the data

For replicability of my research I list the queries used for the draft 
research.

Finding the amount of real users who have reported their location and the
number of countries contained in the data set:

```sql
SELECT COUNT(*) AS users, COUNT(DISTINCT country_code) AS countries
FROM users
WHERE location IS NOT NULL
  AND (country_code IS NOT NULL AND state IS NOT NULL AND city IS NOT NULL)
  AND deleted=0
  AND fake=0
  AND type='USR'
```

Finding the average amount of original (non-forked) repositories per user who
have reported their location:

```sql
SELECT AVG(rcount) FROM (
  SELECT COUNT(p.id) AS rcount
  FROM projects p
  JOIN users u ON p.owner_id = u.id
    AND u.location IS NOT NULL
    AND u.country_code IS NOT NULL AND u.state IS NOT NULL AND u.city IS NOT NULL
    AND u.deleted = 0 AND u.fake = 0 AND u.type = 'USR'
  WHERE p.deleted = 0 AND p.forked_from IS NULL
  GROUP BY p.owner_id
) AS a
```

Bugs
----

For any bug or request, please [create an
issue](https://github.com/Hjdskes/License-locations/issues/new) on [GitHub][github].

License
-------

Please see [LICENSE](https://github.com/Hjdskes/License-locations/blob/master/LICENSE) on [GitHub][github].

**Copyright © 2017** Jente Hidskes &lt;hjdskes@gmail.com&gt;

  [github]: https://github.com/Hjdskes/License-locations
