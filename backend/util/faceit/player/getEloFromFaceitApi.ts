import { SECRETS } from '../../../config/env'

import type faceitPlayerReponse from '../../../@types/player'

const getFaceitPlayer = async (username: string, id?: string): Promise<faceitPlayerReponse> => {
  const path: string = id == undefined ? `/players?nickname=${username}` : `${isFaceitID(id) ? `/players/${id}` : `/players?game=csgo&game_player_id=${id}`}`
  const url: string = SECRETS.faceit.baseUrl + path
  const responseRaw: Response = await fetch(url, {
    method: 'GET',
    headers: SECRETS.faceit.header
  })

  const player: faceitPlayerReponse = await responseRaw.json()
  console.log(player)
  return player
}

const isFaceitID = (id: string): boolean => {
  return new RegExp(SECRETS.regex.faceitId).test(id)
}

export { getFaceitPlayer }
