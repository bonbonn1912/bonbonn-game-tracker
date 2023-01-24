import mongoose from "mongoose"
import { playerSchema } from "./playerSchema"
import { webHookBodySchema } from "./webHookBodySchema"

const extendedPlayerSchema = new mongoose.Schema({
    created_at : { type: Date, required: true, default: Date.now },
    ...playerSchema.obj
    
})
const extendedMatchroomSchema = new mongoose.Schema({
    meta : {
        inserted_at: { type: String, default : () => new Date},
        timestamp: { type: String, default : () => new Date},
        isRunning: { type: Boolean, default: false}
    },
    ...webHookBodySchema.obj
})

export { extendedPlayerSchema, extendedMatchroomSchema}