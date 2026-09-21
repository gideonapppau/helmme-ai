param([string]$Api = "http://localhost:8081")
# Eval runner: each query must surface its expected title in the top 3.
# Exit 0 = all pass. Any failure lists what broke. See seed-queries.md.
$cases = @(
  @("Flutterwave", "Gideon_Appau_CV.pdf"),
  @("anything about marketmate", "Gideon_Appau_CV.pdf"),
  @("Takoradi", "Gideon_Appau_CV.pdf"),
  @("Lighthouse", "Gideon_Appau_CV.pdf"),
  @("Stripe", "Gideon_Appau_CV.pdf"),
  @("Angular", "Gideon_Appau_CV.pdf"),
  @("Clink", "Gideon_Appau_CV.pdf"),
  @("Expo", "Gideon_Appau_CV.pdf"),
  @("ALX", "Gideon_Appau_CV.pdf"),
  @("iCODE", "Gideon_Appau_CV.pdf"),
  @("Maestro", "Gideon_Appau_CV.pdf"),
  @("pooling helmme", "pool-test.pdf"),
  @("quick-save-test", "https://example.com/quick-save-test"),
  @("pgvector guide", "Pgvector guide"),
  @("Go routers", "Go routers"),
  @("remembers why things matter", "helmme remembers why things matter"),
  @("what did I learn about postgres", "pool-test.pdf"),
  @("tell me about payment work", "Gideon_Appau_CV.pdf"),
  @("databases", "Gideon_Appau_CV.pdf"),
  @("offline mobile", "Gideon_Appau_CV.pdf")
)
$pass = 0; $fail = 0
foreach ($c in $cases) {
  try {
    $r = Invoke-RestMethod "$Api/v1/search" -Method POST `
      -Body (@{query = $c[0]} | ConvertTo-Json) -ContentType 'application/json'
    $top3 = @($r.hits | Select-Object -First 3 | ForEach-Object { $_.title })
    if ($top3 -contains $c[1]) { $pass++ } else { $fail++; Write-Output "FAIL: '$($c[0])' wanted '$($c[1])' got [$($top3 -join ' | ')]" }
  } catch { $fail++; Write-Output "FAIL: '$($c[0])' errored: $($_.Exception.Message)" }
}
Write-Output "eval: $pass passed, $fail failed, recall@3 = $([math]::Round($pass / ($pass + $fail), 2))"
exit $fail
