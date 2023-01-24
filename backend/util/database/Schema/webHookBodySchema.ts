import webHookBody, { type payload, type team, type player } from '../../../@types/webhook'
import mongoose from 'mongoose'

const playerSchema = new mongoose.Schema<player>({
  id: String,
  nickname: String,
  avatar: String,
  game_id: String,
  game_name: String,
  game_skill_level: Number,
  membership: String,
  anticheat_required: Boolean
})

const teamSchema = new mongoose.Schema<team>({
  id: String,
  name: String,
  type: String,
  avatar: String,
  leader_id: String,
  co_leader_id: String,
  roster: [playerSchema]
})
const entitySchema = new mongoose.Schema({
  id: String,
  name: String,
  type: String
})

const payLoadSchema = new mongoose.Schema<payload>({
  id: String,
  organizer_id: String,
  region: String,
  game: String,
  version: Number,
  created_at: String,
  updated_at: String,
  started_at: String,
  finished_at: String,
  entity: entitySchema,
  teams: [teamSchema]
})

const webHookBodySchema = new mongoose.Schema({
  transaction_id: String,
  event: { type: String, enum: ['match_status_created', 'match_status_configuring', 'match_status_finished', 'match_status_cancelled'] },
  event_id: String,
  third_party_id: String,
  app_id: String,
  timestamp: String,
  retry_count: Number,
  version: Number,
  payload: payLoadSchema
})

export { webHookBodySchema }
