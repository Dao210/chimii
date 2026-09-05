import { describe, it, expect } from "vitest";
import { LoginResponseSchema } from "./schemas";
describe("login response boundary", () => {
  it("requires both a token and a usable account identity", () => {
    for (const raw of [{}, {token: "", user: {id: "user"}}, {token: "token", user: {id: ""}}, {token: "token", user: {}}, {token: 123, user: {id: "user"}}]) {
      expect(LoginResponseSchema.safeParse(raw).success).toBe(false);
    }
  });
  it("allows additive server fields and defaults nonessential profile data", () => {
    const login = LoginResponseSchema.parse({token: "token", user: {id: "user", future_flag: true}});
    expect(login.user.id).toBe("user");
    expect(login.user.name).toBe("");
  });
});
