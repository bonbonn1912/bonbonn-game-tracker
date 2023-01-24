import faceitPlayerReponse from '../../../@types/player'
import faceitElo from '../../../@types/level'
import mongoose from 'mongoose'

const faceitEloSchema = new mongoose.Schema<faceitElo>({
  username: String,
  isMax: Boolean,
  level: Number,
  elo: Number,
  eloToNextLevel: Number,
  responseString: String
})

const csgoSchema = new mongoose.Schema({
  region: String,
  game_player_id: String,
  skill_level: Number,
  faceit_elo: Number,
  game_player_name: String
})
const gamesSchema = new mongoose.Schema({
  csgo: csgoSchema
})

const platformSchema = new mongoose.Schema({
  steam: String
})
const playerSchema = new mongoose.Schema<faceitPlayerReponse>({
  insertType: { type: String, required: true },
  local: { type: faceitEloSchema, required: true },
  player_id: { type: String, required: true },
  nickname: String,
  avatar: String,
  country: String,
  cover_image: String,
  platforms: platformSchema,
  games: gamesSchema,
  matchHistory: [{ type: String, required: false }]
})

export { playerSchema }
