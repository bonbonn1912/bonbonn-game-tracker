import { SECRETS } from "../config/env"

const validFaceitUsername = (username: string): boolean => {
    let regexp = new RegExp(SECRETS.regex.faceitUsername)
    return regexp.test(username)
}

const validSteam64Id = (steam64id: string): boolean => {
    let regexp = new RegExp(SECRETS.regex.steam64Id)
    return regexp.test(steam64id)
}

const validFaceitId = (faceitid: string): boolean => {
    let regexp = new RegExp(SECRETS.regex.faceitId)
    return regexp.test(faceitid)
}

const LimitParameter = (request: any) : boolean =>{
    return isNaN(request);
}

export {validFaceitUsername, LimitParameter, validSteam64Id, validFaceitId}