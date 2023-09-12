import { SECRETS } from '../../../config/env'

import { type faceitMatchHistory } from '../../../@types/player'
import log from '../../logging/print'

const getLastFiveGames = async (faceitId: string): Promise<string[]> => {
  const url: string = SECRETS.faceit.baseUrl + `/players/${faceitId}/stats/csgo`
  log(url)
  const responseRaw: Response = await fetch(url, {
    method: 'GET',
    headers: SECRETS.faceit.header
  })
  const lastFive: faceitMatchHistory = await responseRaw.json()
  return lastFive.lifetime['Recent Results']
}

export { getLastFiveGames }
