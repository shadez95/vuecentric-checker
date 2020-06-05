# vuecentric-checker
vuecentric-checker ensures VueCentric updater service is running for a list of Windows computers.

## Installation

Use git to clone the repo and then run `go build` inside the project folder. This will build a portable binary. If you have a go environment setup, just run `go install` inside the project directory.

## Usage

The program has one requirement: `-computerList`. Set `-computerList` to a list of computer names in a file.

Example:

```bash
vuecentric-checker -computerList ./ListOfComputers.txt
```

This program can also be used to keep track of what computers are not starting the VueCentric Updater service. It does this by storing data into a sqlite3 database file. This is what `-dbPath` (or `-db`) is for. Setting `-dbPath` to a sqlite3 db file on your computer will save the data there. It stores errors into 2 different tables that represent a list of errors and computers. [GORM](https://gorm.io) is used to model the data and save the data. To see hwo the data is modeled, see `db.go`.

Example:

```bash
vuecentric-checker -computerList /path/to/file/ListOfComputers.txt -db /path/to/sql/file/sqlite3.db
```

## Contribute
I'm always open to contributions. Create a pull request or submit an issue to let me know what you want to contribute. Any help to improve this is appreciated.
