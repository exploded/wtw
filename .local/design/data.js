/* Shows: real titles + TMDB poster paths (image.tmdb.org/t/p/w500/<path>)
   Format: [id, title, year, poster_path, genre]
*/
window.SHOWS = [
  ["breaking-bad", "Breaking Bad", "2008", "/ggFHVNu6YYI5L9pCfOacjizRGt.jpg", "Crime drama"],
  ["the-sopranos", "The Sopranos", "1999", "/rTc7ZXdroqjkKivFPvCPX0Ru7uw.jpg", "Crime drama"],
  ["the-wire", "The Wire", "2002", "/4lbclFySvugI51fwsyxBTOm4DqK.jpg", "Crime drama"],
  ["mad-men", "Mad Men", "2007", "/7Hf0XpiXoCRGmtg9KsCjFt1qOQM.jpg", "Period drama"],
  ["the-leftovers", "The Leftovers", "2014", "/jdZecZmFn1xX2Rh32yT9bWiK4w0.jpg", "Drama"],
  ["true-detective", "True Detective", "2014", "/g0Z2cpFXEmlRkpMS4dSKdjfSPNn.jpg", "Crime"],
  ["severance", "Severance", "2022", "/lFf6LLrQjYldcZItzOkGmMMigP7.jpg", "Sci-fi"],
  ["succession", "Succession", "2018", "/7HW47XbkNQ5fiwQFYGWdw9gs144.jpg", "Drama"],
  ["the-bear", "The Bear", "2022", "/zPyHyJV5lvjkvMyLDZj1RLyq0Q4.jpg", "Comedy drama"],
  ["fleabag", "Fleabag", "2016", "/4ZocdxnOO6q2UbdKye2wgofLFhB.jpg", "Comedy"],
  ["chernobyl", "Chernobyl", "2019", "/hlLXt2tOPT6RRnjiUmoxyG1LTFi.jpg", "Limited series"],
  ["the-crown", "The Crown", "2016", "/1M876KPjulVwppEpldhdc8V4o68.jpg", "Period drama"],
  ["stranger-things", "Stranger Things", "2016", "/49WJfeN0moxb9IPfGn8AIqMGskD.jpg", "Sci-fi"],
  ["game-of-thrones", "Game of Thrones", "2011", "/1XS1oqL89opfnbLl8WnZY1O1uJx.jpg", "Fantasy"],
  ["the-office", "The Office", "2005", "/qWnJzyZhyy74gjpSjIXWmuk0ifX.jpg", "Comedy"],
  ["better-call-saul", "Better Call Saul", "2015", "/fC2HDm5t0kHl7mTm7jxMR31b7by.jpg", "Crime drama"],
  ["watchmen", "Watchmen", "2019", "/5iRDLgCZ5Wr7DGqYqQwkGhcTVaw.jpg", "Sci-fi"],
  ["lost", "Lost", "2004", "/og6S0aTZU6YUJAbqxeKjCa3kY1E.jpg", "Mystery"],
  ["the-americans", "The Americans", "2013", "/srPiDEbNvxinRWDp8rFC8TyrVQv.jpg", "Drama"],
  ["deadwood", "Deadwood", "2004", "/4yWcCIRnsfvOFNNLhKnUv7y91j8.jpg", "Western"],
  ["band-of-brothers", "Band of Brothers", "2001", "/c0Ompvy4ngDoQVuvA5DhHFrfh1l.jpg", "War drama"],
  ["the-knick", "The Knick", "2014", "/zIfQz3VlOl7NnG3ATymJ9SQdLFl.jpg", "Period drama"],
  ["halt-and-catch-fire", "Halt and Catch Fire", "2014", "/A5hjEHa8GbKsPvzs4Hsr7DLuT3X.jpg", "Period drama"],
  ["the-expanse", "The Expanse", "2015", "/khgZjLNoZeuMvJyVcb0YfEPv8YH.jpg", "Sci-fi"],
  ["dark", "Dark", "2017", "/apbrbWs8M9lyOpJYU5WXrpFbk1Z.jpg", "Sci-fi"],
  ["fargo", "Fargo", "2014", "/6gpIQQww0EiloqL5ehB3FkDTWFd.jpg", "Crime"],
  ["barry", "Barry", "2018", "/x5g0frYpVByFxxQdN3GYCLtv3RF.jpg", "Dark comedy"],
  ["atlanta", "Atlanta", "2016", "/v7T6tQVBYlJlpr5jM4OCIFQ3gqJ.jpg", "Comedy drama"],
  ["the-handmaids-tale", "The Handmaid's Tale", "2017", "/oGJQhOpT8S1M56tvSsbEBePV5O1.jpg", "Drama"],
  ["bojack-horseman", "BoJack Horseman", "2014", "/usEpJp8h44EtIYdvhLqMSAWnEhk.jpg", "Animated"],
  ["arrested-development", "Arrested Development", "2003", "/3JqEWFSnWuJTGxbwTfnmPXFwjUv.jpg", "Comedy"],
  ["parks-and-rec", "Parks and Recreation", "2009", "/2hAvsJgi7sN5fvfu4GKf0Z2NSGQ.jpg", "Comedy"],
  ["community", "Community", "2009", "/hZSJVzvXPRBJBvWpz1IDRMFwvCv.jpg", "Comedy"],
  ["it-crowd", "The IT Crowd", "2006", "/zcafUWjyhUqsglmNw9Qlhk5pXVs.jpg", "Comedy"],
  ["six-feet-under", "Six Feet Under", "2001", "/yKx6RkIbXoTfQzMm1yqaUcFxKm5.jpg", "Drama"],
  ["the-shield", "The Shield", "2002", "/sRsJ5SZQUjN5z0vvK5W8CGsXJYW.jpg", "Crime"],
  ["mr-robot", "Mr. Robot", "2015", "/oKIBhzZzDX07SoE2bOLhq2EE8rf.jpg", "Thriller"],
  ["westworld", "Westworld", "2016", "/8MfgyFHf7XEboZJPZXCIDqqiz6e.jpg", "Sci-fi"],
  ["arcane", "Arcane", "2021", "/fqldf2t8ztc9aiwn3k6mlX3tvRT.jpg", "Animated"],
  ["the-mandalorian", "The Mandalorian", "2019", "/eU1i6eHXlzMOlEq0ku1Rzq7Y4wA.jpg", "Sci-fi"],
  ["yellowstone", "Yellowstone", "2018", "/fvmoFP5EcWHjeI1xa4iaHcifkqN.jpg", "Drama"],
  ["ted-lasso", "Ted Lasso", "2020", "/c3OL2eTitmJ2pQQu37Z9LtxXWQA.jpg", "Comedy"],
  ["andor", "Andor", "2022", "/cRuk9GBquxL0jXXHKw1XHlCVbFL.jpg", "Sci-fi"],
  ["house-of-the-dragon", "House of the Dragon", "2022", "/7QMsOTMUswlwxJP0rTTZfmz2tX2.jpg", "Fantasy"],
  ["the-last-of-us", "The Last of Us", "2023", "/uKvVjHNqB5VmOrdxqAt2F7J78ED.jpg", "Drama"],
  ["shogun", "Shōgun", "2024", "/7O4iVfOMQmdCSxhOg1WnzG1AvqM.jpg", "Period drama"],
  ["slow-horses", "Slow Horses", "2022", "/c8uGHwkR1okFMzGy1NWrELN7dFd.jpg", "Spy drama"],
  ["peaky-blinders", "Peaky Blinders", "2013", "/vUUqzWa2LnHIVqkaKVlVGkVcZIW.jpg", "Crime"],
  ["sherlock", "Sherlock", "2010", "/7WTsnHkbA0FaG6R9twfFde0I9hl.jpg", "Mystery"],
  ["the-white-lotus", "The White Lotus", "2021", "/gP7OdAhLUYL0AEt0c1vejRnFV0v.jpg", "Drama"]
];

/* Pre-populated ratings for the demo session */
window.DEMO_RATINGS = {
  "the-sopranos": "liked",
  "the-wire": "liked",
  "mad-men": "liked",
  "true-detective": "liked",
  "severance": "liked",
  "the-leftovers": "liked",
  "fleabag": "liked",
  "chernobyl": "liked",
  "better-call-saul": "liked",
  "game-of-thrones": "disliked",
  "the-mandalorian": "disliked",
  "yellowstone": "disliked",
  "stranger-things": "unseen",
  "the-crown": "unseen",
  "the-bear": "unseen"
};

/* Recommendations rows (curated for the demo) */
window.RECS_TONIGHT = ["dark", "the-knick", "halt-and-catch-fire", "atlanta", "barry", "the-americans"];
window.RECS_SEVERANCE = ["dark", "mr-robot", "westworld", "the-expanse", "watchmen", "andor"];
window.RECS_COMFORT = ["parks-and-rec", "community", "ted-lasso", "fleabag", "the-bear", "arrested-development"];

/* Top liked posters for profile */
window.PROFILE_TOP = ["the-sopranos", "the-wire", "mad-men", "severance", "fleabag"];
