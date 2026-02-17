using System.Collections.Generic;
using System.Threading.Tasks;

namespace BrandishBot.Client
{
    public partial class BrandishBotClient
    {
        #region Predictions

        /// <summary>
        /// Process a prediction outcome from Twitch/YouTube
        /// </summary>
        public async Task<PredictionResult> ProcessPredictionOutcome(
            string platform,
            PredictionWinner winner,
            int totalPointsSpent,
            List<PredictionParticipant> participants)
        {
            return await PostAsync<PredictionResult>("/api/v1/prediction", new
            {
                platform = platform,
                winner = winner,
                total_points_spent = totalPointsSpent,
                participants = participants
            });
        }

        #endregion
    }
}
