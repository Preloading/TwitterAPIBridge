package main

// I will most likely need to store auth tokens in a database, as I can only really get one auth related value from the user, and I can't change that value.
// I will probably send the oauth token as:
// {user did}/{generated uuid}/{encryptionkey}

// Since the only way to detect if a user has logged out is "/1/account/push_destinations/destroy.xml", I do not have a reliable way to detect if a user has logged out.
// I also don't wanna think what would happen if someone found my DB and stole it. Then a bunch of people would have their auth tokens stolen. Having the encryption key
// lets me store the auth tokens where it's encrypted.

// This means that:
// 1. If they sign out, their token on the DB will be unretirevable.
// 2. If someone steals the DB, they can't do anything with the auth tokens.
// 3. Users have better piece of mind knowing that we can't access their bluesky account whenever.

// Is this overcomplicating things? Yes. But I think it's a good idea.

// I will probably use GORM for the DB aswell. Lets me use SQLite while testing, and MySQL/MarinaDB for prod.
