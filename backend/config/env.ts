import * as dotenv from 'dotenv'
import path from 'path'
const NODE_ENV: string = process.env.NODE_ENV != undefined ? process.env.NODE_ENV : ""
let envPath = path.join(__dirname,`./../../${NODE_ENV}.env` )
console.log(envPath)
dotenv.config({path: envPath})

export const SECRETS = {
    PORT: process.env.PORT,
    env: NODE_ENV,
    var1: process.env.test
}