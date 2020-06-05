# vuecentric-checker
vuecentric-checker ensures VueCentric updater service is running for a list of Windows computers.

The program has one requirement, `-computerList`. `-computerList` is a list of computer names in a file.

This program can also be used to keep track of what computers are not starting the VueCentric Updater service. It does this by storing data into a sqlite3 database file. This is what `-dbPath` (or `-db`) is for. Setting `-dbPath` to a sqlite3 db file on your computer will save the data there. It stores errors into 2 different tables that represent a list of errors and computers. [GORM](https://gorm.io) is used to model the data and save the data. To see hwo the data is modeled, see `db.go`.

## Installation
Use git to clone the repo and then run `go build` inside the project folder. This will build a portable binary. If you have a go environment setup, just run `go install` inside the project directory.
