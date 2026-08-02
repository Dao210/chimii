export function GET(request: Request) {
  return Response.redirect(new URL("/logo.svg", request.url), 308);
}
