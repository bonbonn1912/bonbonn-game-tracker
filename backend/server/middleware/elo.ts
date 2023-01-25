import { type Request, type Response, type NextFunction } from 'express'

import type faceitPlayerReponse from '../../@types/player'
import type faceitElo from '../../@types/level'
import { getFaceitPlayer } from '../../util/faceit/player/getEloFromFaceitApi'
import { getLastFiveGames } from '../../util/faceit/player/getLastFiveGames'
import { faceitRankInformation } from '../../util/faceit/getEloInformation'
import { eloReponse, appendHistory, appendWinRate } from '../../util/createResponseString'
import { getCustomFaceitHistory } from '../../util/faceit/player/getCustomHistory'
import { type matchHistory } from '../../@types/customHistory'
import { isWinnerFaction } from '../../util/isWinnerFaction'

const getFaceitElo = async (req: Request, res: Response, next: NextFunction) => {
  try {
    const player: faceitPlayerReponse = await getFaceitPlayer(req.query.username as string, req.query.id as string)
    const rankInformation: faceitElo = faceitRankInformation(player)
    rankInformation.responseString = eloReponse(rankInformation)
    player.local = rankInformation
    req.player = player
    next()
  } catch {
    res.status(200).send('Invalid username')
  }
}

const getLastFive = async (req: Request, res: Response, next: NextFunction) => {
  try {
    const lastFiveGames: string[] = await getLastFiveGames(req.player.player_id)
    req.player.local.responseString += appendHistory(lastFiveGames, '. Last 5 Games: ')
    req.player.matchHistory = lastFiveGames
  } catch {
    req.player.matchHistory = ['']
  }
  next()
}

const getCustomHistory = async (req: Request, res: Response, next: NextFunction) => {
  try {
    const customHistory: matchHistory = await getCustomFaceitHistory(req.player.player_id, req.query.limit as any)
    const matchHistory: string[] = []
    customHistory.items.forEach(singleGame => {
      const winnerFaction = singleGame.results.winner == 'faction1' ? singleGame.teams.faction1 : singleGame.teams.faction2

      if (isWinnerFaction(req.player.player_id, winnerFaction)) {
        matchHistory.push('1')
      } else {
        matchHistory.push('0')
      }
    })
    if (matchHistory.length > 3) {
      req.player.matchHistory = matchHistory.reverse()
      req.player.local.responseString = appendHistory(matchHistory, `${req.player.nickname}'s last ${req.query.limit} Games: `) + appendWinRate(matchHistory)
      next()
    } else {
      res.status(200).send('Play at least 3 Games to see your history')
    }
  } catch {
    res.status(404).send('Something went wrong')
  }
}

export { getFaceitElo, getLastFive, getCustomHistory }
