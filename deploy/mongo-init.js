// Runs only when MongoDB initializes an empty data volume.
const databaseName = process.env.MONGO_DB_NAME || 'bonbonn';
const username = process.env.MONGO_APP_USERNAME;
const password = process.env.MONGO_APP_PASSWORD;
if (!username || !password) {
  throw new Error('MONGO_APP_USERNAME and MONGO_APP_PASSWORD are required');
}
const applicationDatabase = db.getSiblingDB(databaseName);
if (!applicationDatabase.getUser(username)) {
  applicationDatabase.createUser({
    user: username,
    pwd: password,
    roles: [{ role: 'readWrite', db: databaseName }]
  });
}
