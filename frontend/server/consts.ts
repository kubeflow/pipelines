// The hack solution I found for solving:
// Responses fail randomly half way in a partial state, with the last sentence
// being: "Error occurred while trying to proxy request ..."
// Reference: https://github.com/chimurai/http-proxy-middleware/issues/171#issuecomment-439020716
export const HACK_FIX_HPM_PARTIAL_RESPONSE_HEADERS = {
  Connection: 'keep-alive',
};
