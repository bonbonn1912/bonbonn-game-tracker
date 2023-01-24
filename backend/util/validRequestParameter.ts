import { SECRETS } from '../config/env'

const validFaceitUsername = (username: string): boolean => {
  const regexp = new RegExp(SECRETS.regex.faceitUsername)
  return regexp.test(username)
}

const validSteam64Id = (steam64id: string): boolean => {
  const regexp = new RegExp(SECRETS.regex.steam64Id)
  return regexp.test(steam64id)
}

const validFaceitId = (faceitid: string): boolean => {
  const regexp = new RegExp(SECRETS.regex.faceitId)
  return regexp.test(faceitid)
}

const validLimit = (request: any): boolean => {
  if (!isNaN(request) && parseInt(request) > 3 && parseInt(request) < 101) {
    return true
  }
  return false
}

export { validFaceitUsername, validSteam64Id, validFaceitId, validLimit }
