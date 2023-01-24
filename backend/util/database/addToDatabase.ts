import mongoose, { Schema } from "mongoose";
import faceitPlayerReponse from "../../@types/player";
import { extendedPlayerSchema, extendedMatchroomSchema} from "./Schema/extendedSchema";
import { SECRETS } from "../../config/env";
import { InsertType } from "../../@types/insertTypes";
import webHookBody from "../../@types/webhook";

const addPlayerToDB = async (player: faceitPlayerReponse, insertType: InsertType) => {
    player.insertType = insertType
    mongoInsert(player, extendedPlayerSchema, SECRETS.mongo.playerCollectionName as string);
};

const addMatchroomToDB = async (matchroom: webHookBody ) =>{
    mongoInsert(matchroom, extendedMatchroomSchema, SECRETS.mongo.matchRoomCollectionName as string);
}

const getModel = (schema: Schema, collection: string) =>{
    return mongoose.models[collection] || mongoose.model(collection, schema)
}

const mongoInsert = async (document: webHookBody | faceitPlayerReponse,schema: Schema, collection: string) =>{
    mongoose.set('strictQuery', true)
    let connectionString: string = SECRETS.mongo.connectionString as string +SECRETS.mongo.dbName as string
    const Model = getModel(schema, collection)
    const entry = new Model(document)
    let connectResult;
    try{
         connectResult = await mongoose.connect(connectionString,SECRETS.mongo.timeoutAfter);
    }catch{
        console.log("Could not connect to Database")
    }
    
    if(connectResult != undefined){
        await entry.save((err: Error) =>{
            if(err){
                console.log(err)
            }else{
                console.log(`Added Entry to collection ${collection}`)
            }
            mongoose.disconnect()
        })     
    } 
}

export { addPlayerToDB, addMatchroomToDB};
