import { SECRETS } from '../../../env'

import { type faceitMatchHistory } from '../../../@types/player'

const getLastFiveGames = async (faceitId: string): Promise<string[]> => {
  const url: string = SECRETS.faceit.baseUrl + `/players/${faceitId}/stats/csgo`
  console.log(url)
  const responseRaw: Response = await fetch(url, {
    method: 'GET',
    headers: SECRETS.faceit.header
  })
  const lastFive: faceitMatchHistory = await responseRaw.json()
  return lastFive.lifetime['Recent Results']
}

export { getLastFiveGames }
