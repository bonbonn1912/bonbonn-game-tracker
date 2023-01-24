import * as dotenv from 'dotenv'
import path from 'path'
const NODE_ENV: string = process.env.NODE_ENV == 'dev' ? process.env.NODE_ENV : 'production'
const envPath: string = path.join(__dirname, `./../../${NODE_ENV}.env`)

dotenv.config({ path: envPath })

export const SECRETS = {
  PORT: process.env.PORT,
  env: NODE_ENV,
  var1: process.env.test,
  faceit: {
    apiKey: process.env.apiKey,
    baseUrl: 'https://open.faceit.com/data/v4',
    header: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
      Authorization: 'Bearer ' + process.env.apiKey
    }
  },
  regex: {
    steam64Id: '^[0-9]{17}$',
    faceitId: '^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$',
    faceitUsername: '^[a-zA-Z0-9_-]{3,12}$'
  },
  mongo: {
    connectionString: process.env.mongoString,
    dbName: process.env.mongoDbName,
    playerCollectionName: process.env.mongoPlayerCollectionName,
    matchRoomCollectionName: process.env.mongoMatchroomCollectionName,
    timeoutAfter: { serverSelectionTimeoutMS: 1000 }
  }
}
