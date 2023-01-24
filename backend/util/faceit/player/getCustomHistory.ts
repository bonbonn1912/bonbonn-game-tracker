import { SECRETS } from '../../../config/env'
import { type matchHistory } from '../../../@types/customHistory'

const getCustomFaceitHistory = async (faceitId: string, limit: number): Promise<matchHistory> => {
  const url: string = SECRETS.faceit.baseUrl + `/players/${faceitId}/history?game=csgo&offset=0&limit=${limit}`
  console.log(url)
  const responseRaw: Response = await fetch(url, {
    method: 'GET',
    headers: SECRETS.faceit.header
  })
  const customMatchHistory: matchHistory = await responseRaw.json()
  return customMatchHistory
}

export { getCustomFaceitHistory }
