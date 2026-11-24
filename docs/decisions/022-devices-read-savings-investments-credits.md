# 022 Enrolled devices read savings, investments and credits

Status: accepted and built in the server and phone app

Context: the phone is becoming the household app
([ADR 020](020-the-phone-app-is-the-household-app.md)) and writes where a
movement's money went ([ADR 021](021-details-of-money-moved.md)). To do that it
has to list the savings accounts, the investment accounts with their
instruments and the credits, and to show them on pages of their own.
[ADR 012](012-devices-read-goals.md) opened only the goals to devices and said
each further area is added deliberately, one at a time.

Decision:
- **The savings, investments and credits read routes take either credential**,
  the session cookie or `Authorization: Bearer <device token>`, as the goals
  routes do. That is every `GET` under `/savings`, `/investments` and `/credits`.
- **Every person already reads all of it in the browser** ([ADR 010](010-household-data-is-shared.md)),
  so a device reads nothing new. The answers carry the owner of each account
  (`owner` on a savings account, `user` on an investment account), and the
  phone lists only the accounts of its person and of the household when it
  picks where money goes. The server still decides who can write
  ([ADR 021](021-details-of-money-moved.md)); the list is a convenience.
- **One new read: `GET /credits/due`.** The credits that are not paid off, have
  started, have an expected day and have no regular payment in the current
  month, with the amount of their last regular payment as the instalment. The
  phone shows it as a notice, and a credit contract now carries its
  `expectedDay`.
- **Cashflow movements stay closed to devices.** Only what earlier decisions
  opened can be read from there.

Consequences: a device token reads these household accounts and can write
permitted detail records only through its person's cashflow movements. Revoke
it with `device revoke`.