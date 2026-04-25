package db

import (
	"database/sql"
	"fmt"
)

type SeedShow struct {
	ID         string
	Title      string
	Year       string
	PosterPath string
	Genre      string
	Popularity float64
}

var SeedShows = []SeedShow{
	{"breaking-bad", "Breaking Bad", "2008", "/ggFHVNu6YYI5L9pCfOacjizRGt.jpg", "Crime drama", 100},
	{"the-sopranos", "The Sopranos", "1999", "/rTc7ZXdroqjkKivFPvCPX0Ru7uw.jpg", "Crime drama", 99},
	{"the-wire", "The Wire", "2002", "/4lbclFySvugI51fwsyxBTOm4DqK.jpg", "Crime drama", 98},
	{"mad-men", "Mad Men", "2007", "/7Hf0XpiXoCRGmtg9KsCjFt1qOQM.jpg", "Period drama", 97},
	{"the-leftovers", "The Leftovers", "2014", "/jdZecZmFn1xX2Rh32yT9bWiK4w0.jpg", "Drama", 96},
	{"true-detective", "True Detective", "2014", "/g0Z2cpFXEmlRkpMS4dSKdjfSPNn.jpg", "Crime", 95},
	{"severance", "Severance", "2022", "/lFf6LLrQjYldcZItzOkGmMMigP7.jpg", "Sci-fi", 94},
	{"succession", "Succession", "2018", "/7HW47XbkNQ5fiwQFYGWdw9gs144.jpg", "Drama", 93},
	{"the-bear", "The Bear", "2022", "/zPyHyJV5lvjkvMyLDZj1RLyq0Q4.jpg", "Comedy drama", 92},
	{"fleabag", "Fleabag", "2016", "/4ZocdxnOO6q2UbdKye2wgofLFhB.jpg", "Comedy", 91},
	{"chernobyl", "Chernobyl", "2019", "/hlLXt2tOPT6RRnjiUmoxyG1LTFi.jpg", "Limited series", 90},
	{"the-crown", "The Crown", "2016", "/1M876KPjulVwppEpldhdc8V4o68.jpg", "Period drama", 89},
	{"stranger-things", "Stranger Things", "2016", "/49WJfeN0moxb9IPfGn8AIqMGskD.jpg", "Sci-fi", 88},
	{"game-of-thrones", "Game of Thrones", "2011", "/1XS1oqL89opfnbLl8WnZY1O1uJx.jpg", "Fantasy", 87},
	{"the-office", "The Office", "2005", "/qWnJzyZhyy74gjpSjIXWmuk0ifX.jpg", "Comedy", 86},
	{"better-call-saul", "Better Call Saul", "2015", "/fC2HDm5t0kHl7mTm7jxMR31b7by.jpg", "Crime drama", 85},
	{"watchmen", "Watchmen", "2019", "/5iRDLgCZ5Wr7DGqYqQwkGhcTVaw.jpg", "Sci-fi", 84},
	{"lost", "Lost", "2004", "/og6S0aTZU6YUJAbqxeKjCa3kY1E.jpg", "Mystery", 83},
	{"the-americans", "The Americans", "2013", "/srPiDEbNvxinRWDp8rFC8TyrVQv.jpg", "Drama", 82},
	{"deadwood", "Deadwood", "2004", "/4yWcCIRnsfvOFNNLhKnUv7y91j8.jpg", "Western", 81},
	{"band-of-brothers", "Band of Brothers", "2001", "/c0Ompvy4ngDoQVuvA5DhHFrfh1l.jpg", "War drama", 80},
	{"the-knick", "The Knick", "2014", "/zIfQz3VlOl7NnG3ATymJ9SQdLFl.jpg", "Period drama", 79},
	{"halt-and-catch-fire", "Halt and Catch Fire", "2014", "/A5hjEHa8GbKsPvzs4Hsr7DLuT3X.jpg", "Period drama", 78},
	{"the-expanse", "The Expanse", "2015", "/khgZjLNoZeuMvJyVcb0YfEPv8YH.jpg", "Sci-fi", 77},
	{"dark", "Dark", "2017", "/apbrbWs8M9lyOpJYU5WXrpFbk1Z.jpg", "Sci-fi", 76},
	{"fargo", "Fargo", "2014", "/6gpIQQww0EiloqL5ehB3FkDTWFd.jpg", "Crime", 75},
	{"barry", "Barry", "2018", "/x5g0frYpVByFxxQdN3GYCLtv3RF.jpg", "Dark comedy", 74},
	{"atlanta", "Atlanta", "2016", "/v7T6tQVBYlJlpr5jM4OCIFQ3gqJ.jpg", "Comedy drama", 73},
	{"the-handmaids-tale", "The Handmaid's Tale", "2017", "/oGJQhOpT8S1M56tvSsbEBePV5O1.jpg", "Drama", 72},
	{"bojack-horseman", "BoJack Horseman", "2014", "/usEpJp8h44EtIYdvhLqMSAWnEhk.jpg", "Animated", 71},
	{"arrested-development", "Arrested Development", "2003", "/3JqEWFSnWuJTGxbwTfnmPXFwjUv.jpg", "Comedy", 70},
	{"parks-and-rec", "Parks and Recreation", "2009", "/2hAvsJgi7sN5fvfu4GKf0Z2NSGQ.jpg", "Comedy", 69},
	{"community", "Community", "2009", "/hZSJVzvXPRBJBvWpz1IDRMFwvCv.jpg", "Comedy", 68},
	{"it-crowd", "The IT Crowd", "2006", "/zcafUWjyhUqsglmNw9Qlhk5pXVs.jpg", "Comedy", 67},
	{"six-feet-under", "Six Feet Under", "2001", "/yKx6RkIbXoTfQzMm1yqaUcFxKm5.jpg", "Drama", 66},
	{"the-shield", "The Shield", "2002", "/sRsJ5SZQUjN5z0vvK5W8CGsXJYW.jpg", "Crime", 65},
	{"mr-robot", "Mr. Robot", "2015", "/oKIBhzZzDX07SoE2bOLhq2EE8rf.jpg", "Thriller", 64},
	{"westworld", "Westworld", "2016", "/8MfgyFHf7XEboZJPZXCIDqqiz6e.jpg", "Sci-fi", 63},
	{"arcane", "Arcane", "2021", "/fqldf2t8ztc9aiwn3k6mlX3tvRT.jpg", "Animated", 62},
	{"the-mandalorian", "The Mandalorian", "2019", "/eU1i6eHXlzMOlEq0ku1Rzq7Y4wA.jpg", "Sci-fi", 61},
	{"yellowstone", "Yellowstone", "2018", "/fvmoFP5EcWHjeI1xa4iaHcifkqN.jpg", "Drama", 60},
	{"ted-lasso", "Ted Lasso", "2020", "/c3OL2eTitmJ2pQQu37Z9LtxXWQA.jpg", "Comedy", 59},
	{"andor", "Andor", "2022", "/cRuk9GBquxL0jXXHKw1XHlCVbFL.jpg", "Sci-fi", 58},
	{"house-of-the-dragon", "House of the Dragon", "2022", "/7QMsOTMUswlwxJP0rTTZfmz2tX2.jpg", "Fantasy", 57},
	{"the-last-of-us", "The Last of Us", "2023", "/uKvVjHNqB5VmOrdxqAt2F7J78ED.jpg", "Drama", 56},
	{"shogun", "Shogun", "2024", "/7O4iVfOMQmdCSxhOg1WnzG1AvqM.jpg", "Period drama", 55},
	{"slow-horses", "Slow Horses", "2022", "/c8uGHwkR1okFMzGy1NWrELN7dFd.jpg", "Spy drama", 54},
	{"peaky-blinders", "Peaky Blinders", "2013", "/vUUqzWa2LnHIVqkaKVlVGkVcZIW.jpg", "Crime", 53},
	{"sherlock", "Sherlock", "2010", "/7WTsnHkbA0FaG6R9twfFde0I9hl.jpg", "Mystery", 52},
	{"the-white-lotus", "The White Lotus", "2021", "/gP7OdAhLUYL0AEt0c1vejRnFV0v.jpg", "Drama", 51},
}

func SeedAllShows(database *sql.DB) error {
	for _, s := range SeedShows {
		_, err := database.Exec(
			`INSERT OR IGNORE INTO shows (id, title, year, poster_path, genre, popularity) VALUES (?, ?, ?, ?, ?, ?)`,
			s.ID, s.Title, s.Year, s.PosterPath, s.Genre, s.Popularity,
		)
		if err != nil {
			return fmt.Errorf("seeding show %s: %w", s.ID, err)
		}
	}
	return nil
}
