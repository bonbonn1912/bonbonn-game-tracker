import * as dotenv from 'dotenv'
import path from 'path'
const NODE_ENV: string = process.env.NODE_ENV == 'dev' ? process.env.NODE_ENV : "production"
let envPath: string = path.join(__dirname,`./../../${NODE_ENV}.env` )

dotenv.config({path: envPath})

export const SECRETS = {
    PORT: process.env.PORT,
    env: NODE_ENV,
    var1: process.env.test,
    faceit : {
        apiKey : process.env.apiKey, 
        baseUrl: "https://open.faceit.com/data/v4",
        header: {
            'Accept': 'application/json',
            'Content-Type': 'application/json',
            'Authorization': 'Bearer '+ process.env.apiKey 
        }
    }
}