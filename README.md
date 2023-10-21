# IBL

IBL is a simple utility to make development of Infinity Bot List easier as well as to allow bot developers to test the API. 

For more information, try running "ibl --help"

If you wish to add a new command, use "cobra-cli add NAME"

## IBLFile types

### db

- db.seed - A file that when loaded seeds a database with optional seed data
- db.backup - A file that when loaded backs up a database as an encrypted section based on a private key. This can then be safely stored on s3 or other storage providers
- db.staging - A sanitized staging file that can then be restored to a staging database.

See ``helper_scripts`` for in production usage of these options for managing our database

**Still a work in progress**
