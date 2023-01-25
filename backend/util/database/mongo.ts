import mongoose, { type Schema } from 'mongoose'
import type faceitPlayerReponse from '../../@types/player'
import { playerSchema } from './Schema/playerSchema'
import { webHookBodySchema } from './Schema/webHookBodySchema'
import { SECRETS } from '../../config/env'
import { type InsertType } from '../../@types/insertTypes'
import type webHookBody from '../../@types/webhook'
import log from '../logging/print'

const addPlayerToDB = async (player: faceitPlayerReponse, insertType: InsertType) => {
  player.insertType = insertType
  mongoInsert(player, extendSchema(playerSchema, null), SECRETS.mongo.playerCollectionName as string)
}

const addMatchroomToDB = async (matchroom: webHookBody) => {
  mongoInsert(matchroom, extendSchema(webHookBodySchema, true), SECRETS.mongo.matchRoomCollectionName as string)
}

const getModel = (schema: Schema, collection: string) => {
  return mongoose.models[collection] || mongoose.model(collection, schema)
}

const getConnection = async () =>{
  let connectionResult;
  const connectionString: string = SECRETS.mongo.connectionString as string + SECRETS.mongo.dbName
  try {
    connectionResult = await mongoose.connect(connectionString, SECRETS.mongo.timeoutAfter)
  } catch {
    log('Could not connect to Database')
  }finally{
    return connectionResult
  }
}

const extendSchema = (schema: mongoose.Schema, isRunning: boolean | null): mongoose.Schema => {
  const extendedSchema = new mongoose.Schema({
    meta: {
      inserted_at: { type: String, default: () => new Date() },
      timestamp: { type: String, default: () => new Date() },
      isRunning: { type: Boolean, default: isRunning, required: false }
    },
    ...schema.obj
  })
  return extendedSchema
}

const mongoInsert = async (document: webHookBody | faceitPlayerReponse, schema: Schema, collection: string) => {
  mongoose.set('strictQuery', true)
  const Model = getModel(schema, collection)
  const entry = new Model(document)
  let connectResult = await getConnection()
  if (connectResult != undefined) {
    await entry.save((err: Error) => {
      if (err) {
        log(err)
      } else {
        log(`Added Entry to collection ${collection}`)
      }
      mongoose.disconnect()
    })
  }
}

const mongoUpdate = async (key: string) =>{
  const filter = { "meta.isRunning" : true , "streamer" : key }
  const update = { "meta.isRunning" : false }
  mongoose.set('strictQuery', true)
  let connectResult = await getConnection()
  const extendedSchema = extendSchema(webHookBodySchema, true)
  const Model = getModel(extendedSchema,SECRETS.mongo.matchRoomCollectionName as string );
  if (connectResult != undefined) {
     let result = await Model.findOneAndUpdate(filter, update)
     if(result == null){
      log("Could not update Game")
     }else{
      log("Game closed in DB")
     } 
     mongoose.disconnect()
  }
 
}

const alreadyLive = async (key: string) : Promise<Boolean> =>{
  const selection = { "meta.isRunning" : true , "streamer" : key }
  mongoose.set('strictQuery', true)
  let connectResult = await getConnection()
  const extendedSchema = extendSchema(webHookBodySchema, true)
  const Model = getModel(extendedSchema,SECRETS.mongo.matchRoomCollectionName as string);
  if(connectResult != undefined){
    let response =  Model.find(selection)
    let matches = await response.lean().exec() as unknown as webHookBody[]
    log(`Live Games for Streamer ${key} : ${matches.length}`)
    if(matches.length > 0){
      return true
    }else{
      return false
    }
  }
  return false
  
}

const dropDB = () =>{
  const connectionString: string = SECRETS.mongo.connectionString as string + SECRETS.mongo.dbName
  mongoose.connect(connectionString, () =>{
    mongoose.connection.db.dropDatabase()
  })
  return true
}

const getLiveGames = async () : Promise<webHookBody[]> =>{
  const selection = { "meta.isRunning" : true }
  mongoose.set('strictQuery', true)
  const connectionString: string = SECRETS.mongo.connectionString as string + SECRETS.mongo.dbName
  let connectResult
  try {
    connectResult = await mongoose.connect(connectionString, SECRETS.mongo.timeoutAfter)
  } catch {
    log('Could not connect to Database')
  }
  const extendedSchema = extendSchema(webHookBodySchema, true)
  const Model = getModel(extendedSchema,SECRETS.mongo.matchRoomCollectionName as string);
  let response =  Model.find(selection)
  let docs = await response.lean().exec() as unknown
  let matches = docs as webHookBody[]
  return matches
}

export { addPlayerToDB, addMatchroomToDB, mongoUpdate, getLiveGames, alreadyLive, dropDB }
