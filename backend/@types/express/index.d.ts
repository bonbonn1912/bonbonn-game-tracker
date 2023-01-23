import faceitPlayerReponse from "../player";

declare global {
  namespace Express {
      export interface Request {
          responseString: string;
      }
  }
}