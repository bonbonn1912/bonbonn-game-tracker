import { Mongoose, Schema, model } from 'mongoose'

export default interface faceitElo {
    username: string,
    isMax: boolean,
    level: number,
    elo: number,
    eloToNextLevel?: number,
    responseString: string,
}
