import webHookBody from '../@types/webhook'
import type faceitElo from '../@types/level'
import { isStreamer, redirectUrl } from './liveGames'

const eloReponse = (rankInformation: faceitElo): string => {
  return `${rankInformation.username}'s Elo:  ${rankInformation.elo} (csgo) ${rankInformation.cs2elo == undefined ? "" :` / ${rankInformation.cs2elo} (cs2)`}`
  
 /* if (rankInformation.isMax) {
    return `${rankInformation.username} is level!!! ${rankInformation.level} with ${rankInformation.elo} elo`
  } else {
    return `${rankInformation.username} is level ${rankInformation.level} with ${rankInformation.elo} elo. Elo needed for level ${rankInformation.level + 1}: ${rankInformation.eloToNextLevel} `
  } */
}

const appendHistory = (matchHistory: string[], firstPart: string): string => {
  if (matchHistory.length == 0) {
    return ''
  }
  const winLose = matchHistory.reverse().map(result => result == '1' ? 'W/' : 'L/')
  const winLoseString = firstPart + winLose.join('')
  return winLoseString.substring(0, winLoseString.length - 1)
}

const appendWinRate = (matchHistory: string[]) => {
  if (matchHistory.length == 0) {
    return ''
  }
  const arraySum: number = matchHistory.map(el => parseInt(el)).reduce((curr: number, prev: number) => curr + prev)
  const winrate: number = (arraySum / matchHistory.length)
  return `. Winrate: ${Math.floor(winrate * 100)}%`
}

const matchRoomResponse = (match: webHookBody, streamer: string) : string =>{
  let lobbyUrl: string | undefined = isStreamer(streamer) ? redirectUrl.get(streamer) : `https://www.faceit.com/en/csgo/room/${match.payload.id}`
  return `${match.matchup.team1.name} (${match.matchup.team1.avgElo}) vs. ${match.matchup.team2.name} (${match.matchup.team2.avgElo}) on ${match.map}. Link zum Matchroom: ${lobbyUrl}`
}

export { eloReponse, appendHistory, appendWinRate, matchRoomResponse }
