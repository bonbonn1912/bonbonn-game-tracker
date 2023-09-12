import { type Request, type Response, type NextFunction } from 'express'
import { validFaceitUsername, validSteam64Id, validFaceitId, validLimit } from '../../util/validRequestParameter'
import { SECRETS } from '../../config/env'

const checkEloInput = (req: Request, res: Response, next: NextFunction) => {
  const username: any = req.query.username

  if (username == undefined || !validFaceitUsername(username)) {
    res.status(200).send("Nicknames can be made of letters (a-z, A-Z), numbers (0-9), '_' and '-' only. 3-12 characters")
    return
  }
  next()
}

const checkEloBySteamIdInput = (req: Request, res: Response, next: NextFunction) => {
  const steam64id: any = req.query.id
  if (steam64id == undefined || !validSteam64Id(steam64id)) {
    res.status(404).send('Steam 64 Ids are made of 17 numbers (0-9)')
    return
  }
  next()
}

const checkEloByFaceitIdInput = (req: Request, res: Response, next: NextFunction) => {
  const faceitId: any = req.query.id
  if (faceitId == undefined || !validFaceitId(faceitId)) {
    res.status(404).send('Faceit Ids must match following pattern : ' + SECRETS.regex.faceitId)
    return
  }
  next()
}

const checkMatchHistoryInput = (req: Request, res: Response, next: NextFunction) => {
  const username: any = req.query.username
  const limit: any = req.query.limit
  if (username == undefined || limit == undefined || !validFaceitUsername(username) || !validLimit(limit)) {
    res.status(200).send('Incorrect faceit username or invalid limit (4-100)')
    return
  }
  next()
}

export { checkEloInput, checkEloBySteamIdInput, checkEloByFaceitIdInput, checkMatchHistoryInput }
