import mongoose, { model} from "mongoose";
import faceitPlayerReponse from "../../@types/player";
import { playerSchema } from "./Schema/playerSchema";
import { SECRETS } from "../../config/env";
import { InsertType } from "../../@types/insertTypes";

const options = { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' };

 
const extendedSchema = new mongoose.Schema({
    created_at : { type: Date, required: true, default: Date.now },
    ...playerSchema.obj
    
})
const addPlayerToDB = async (player: faceitPlayerReponse, insertType: InsertType) => {
   
    mongoose.set('strictQuery', true)
    let connectionString: string = SECRETS.mongo.connectionString as string +SECRETS.mongo.dbName as string
    const Player = model(SECRETS.mongo.playerCollectionName as string, extendedSchema);
    player.insertType = insertType
    const entry = new Player(player)
    let connectResult;
    try{
         connectResult = await mongoose.connect(connectionString,SECRETS.mongo.timeoutAfter);
    }catch{
        console.log(connectResult)
    }
    
    if(connectResult != undefined){
        await entry.save((err) =>{
            if(err){
                console.log("Could not insert Player into db")
            }else{
                console.log("Player inserted")
            }
            mongoose.disconnect()
        })
       
    }

    
    
  
};

export { addPlayerToDB };
