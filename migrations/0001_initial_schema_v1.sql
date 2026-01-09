-- +goose Up
-- BrandishBot v1.0 - Initial Schema
-- Squashed from 29 development migrations (2025-11 through 2026-01)

CREATE TABLE public.crafting_recipes (
    recipe_id integer NOT NULL,
    target_item_id integer NOT NULL,
    base_cost jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT now()
);
CREATE SEQUENCE public.crafting_recipes_recipe_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.crafting_recipes_recipe_id_seq OWNED BY public.crafting_recipes.recipe_id;
CREATE TABLE public.disassemble_outputs (
    output_id integer NOT NULL,
    recipe_id integer NOT NULL,
    item_id integer NOT NULL,
    quantity integer NOT NULL
);
CREATE SEQUENCE public.disassemble_outputs_output_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.disassemble_outputs_output_id_seq OWNED BY public.disassemble_outputs.output_id;
CREATE TABLE public.disassemble_recipes (
    recipe_id integer NOT NULL,
    source_item_id integer NOT NULL,
    quantity_consumed integer DEFAULT 1 NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);
CREATE SEQUENCE public.disassemble_recipes_recipe_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.disassemble_recipes_recipe_id_seq OWNED BY public.disassemble_recipes.recipe_id;
CREATE TABLE public.engagement_metrics (
    id integer NOT NULL,
    user_id character varying(255) NOT NULL,
    metric_type character varying(50) NOT NULL,
    metric_value integer DEFAULT 1,
    recorded_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    metadata jsonb
);
CREATE SEQUENCE public.engagement_metrics_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.engagement_metrics_id_seq OWNED BY public.engagement_metrics.id;
CREATE TABLE public.engagement_weights (
    metric_type character varying(50) NOT NULL,
    weight numeric(5,2) DEFAULT 1.0,
    description text,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE public.events (
    id bigint NOT NULL,
    event_type character varying(100) NOT NULL,
    user_id character varying(100),
    payload jsonb NOT NULL,
    metadata jsonb,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP
);
CREATE SEQUENCE public.events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.events_id_seq OWNED BY public.events.id;
CREATE TABLE public.gamble_opened_items (
    gamble_id uuid,
    user_id uuid,
    item_id integer,
    value bigint NOT NULL
);
CREATE TABLE public.gamble_participants (
    gamble_id uuid NOT NULL,
    user_id uuid NOT NULL,
    lootbox_bets jsonb NOT NULL
);
CREATE TABLE public.gambles (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    initiator_id uuid NOT NULL,
    state text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    join_deadline timestamp with time zone NOT NULL,
    CONSTRAINT gambles_state_check CHECK ((state = ANY (ARRAY['Created'::text, 'Joining'::text, 'Opening'::text, 'Completed'::text, 'Refunded'::text])))
);
CREATE TABLE public.item_type_assignments (
    item_id integer NOT NULL,
    item_type_id integer NOT NULL
);
CREATE TABLE public.item_types (
    item_type_id integer NOT NULL,
    type_name character varying(100) NOT NULL
);
CREATE SEQUENCE public.item_types_item_type_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.item_types_item_type_id_seq OWNED BY public.item_types.item_type_id;
CREATE TABLE public.items (
    item_id integer NOT NULL,
    internal_name character varying(255) NOT NULL,
    item_description text,
    base_value integer DEFAULT 0,
    public_name character varying(100),
    handler character varying(50),
    default_display character varying(255)
);
CREATE SEQUENCE public.items_item_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.items_item_id_seq OWNED BY public.items.item_id;
CREATE TABLE public.job_level_bonuses (
    id integer NOT NULL,
    job_id integer NOT NULL,
    min_level integer NOT NULL,
    bonus_type text NOT NULL,
    bonus_value numeric(10,4) NOT NULL,
    description text
);
CREATE SEQUENCE public.job_level_bonuses_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.job_level_bonuses_id_seq OWNED BY public.job_level_bonuses.id;
CREATE TABLE public.job_xp_events (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    user_id uuid NOT NULL,
    job_id integer NOT NULL,
    xp_amount integer NOT NULL,
    source_type text NOT NULL,
    source_metadata jsonb,
    recorded_at timestamp with time zone DEFAULT now()
);
CREATE TABLE public.jobs (
    id integer NOT NULL,
    job_key text NOT NULL,
    display_name text NOT NULL,
    description text,
    associated_features text[],
    created_at timestamp with time zone DEFAULT now()
);
CREATE SEQUENCE public.jobs_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.jobs_id_seq OWNED BY public.jobs.id;
CREATE TABLE public.link_tokens (
    token character varying(8) NOT NULL,
    source_platform character varying(20) NOT NULL,
    source_platform_id character varying(100) NOT NULL,
    target_platform character varying(20),
    target_platform_id character varying(100),
    state character varying(20) DEFAULT 'pending'::character varying,
    created_at timestamp with time zone DEFAULT now(),
    expires_at timestamp with time zone NOT NULL
);
CREATE TABLE public.moderators (
    moderator_id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);
CREATE TABLE public.platforms (
    platform_id integer NOT NULL,
    name character varying(50) NOT NULL
);
CREATE SEQUENCE public.platforms_platform_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.platforms_platform_id_seq OWNED BY public.platforms.platform_id;
CREATE TABLE public.progression_nodes (
    id integer NOT NULL,
    node_key character varying(100) NOT NULL,
    node_type character varying(50) NOT NULL,
    display_name character varying(200) NOT NULL,
    description text,
    max_level integer DEFAULT 1,
    unlock_cost integer DEFAULT 1000,
    sort_order integer DEFAULT 0,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    tier integer DEFAULT 1 NOT NULL,
    size character varying(20) DEFAULT 'medium'::character varying NOT NULL,
    category character varying(50) DEFAULT 'uncategorized'::character varying NOT NULL
);
CREATE SEQUENCE public.progression_nodes_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.progression_nodes_id_seq OWNED BY public.progression_nodes.id;
CREATE TABLE public.progression_prerequisites (
    node_id integer NOT NULL,
    prerequisite_node_id integer NOT NULL,
    CONSTRAINT progression_prerequisites_check CHECK ((node_id <> prerequisite_node_id))
);
CREATE TABLE public.progression_resets (
    id integer NOT NULL,
    reset_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    reset_by character varying(255),
    reason text,
    nodes_reset_count integer,
    engagement_score_at_reset integer
);
CREATE SEQUENCE public.progression_resets_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.progression_resets_id_seq OWNED BY public.progression_resets.id;
CREATE TABLE public.progression_unlock_progress (
    id integer NOT NULL,
    node_id integer,
    target_level integer,
    contributions_accumulated integer DEFAULT 0 NOT NULL,
    started_at timestamp without time zone DEFAULT now() NOT NULL,
    unlocked_at timestamp without time zone,
    voting_session_id integer
);
CREATE SEQUENCE public.progression_unlock_progress_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.progression_unlock_progress_id_seq OWNED BY public.progression_unlock_progress.id;
CREATE TABLE public.progression_unlocks (
    id integer NOT NULL,
    node_id integer,
    current_level integer DEFAULT 1,
    unlocked_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    unlocked_by character varying(50),
    engagement_score integer DEFAULT 0
);
CREATE SEQUENCE public.progression_unlocks_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.progression_unlocks_id_seq OWNED BY public.progression_unlocks.id;
CREATE TABLE public.progression_voting (
    id integer NOT NULL,
    node_id integer,
    target_level integer DEFAULT 1,
    vote_count integer DEFAULT 0,
    voting_started_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    voting_ends_at timestamp without time zone,
    is_active boolean DEFAULT true
);
CREATE SEQUENCE public.progression_voting_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.progression_voting_id_seq OWNED BY public.progression_voting.id;
CREATE TABLE public.progression_voting_options (
    id integer NOT NULL,
    session_id integer NOT NULL,
    node_id integer NOT NULL,
    target_level integer DEFAULT 1 NOT NULL,
    vote_count integer DEFAULT 0 NOT NULL,
    last_highest_vote_at timestamp without time zone,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);
CREATE SEQUENCE public.progression_voting_options_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.progression_voting_options_id_seq OWNED BY public.progression_voting_options.id;
CREATE TABLE public.progression_voting_sessions (
    id integer NOT NULL,
    started_at timestamp without time zone DEFAULT now() NOT NULL,
    ended_at timestamp without time zone,
    voting_deadline timestamp without time zone DEFAULT (now() + '24:00:00'::interval) NOT NULL,
    winning_option_id integer,
    status character varying(20) DEFAULT 'voting'::character varying NOT NULL,
    created_at timestamp without time zone DEFAULT now() NOT NULL
);
CREATE SEQUENCE public.progression_voting_sessions_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.progression_voting_sessions_id_seq OWNED BY public.progression_voting_sessions.id;
CREATE TABLE public.recipe_associations (
    association_id integer NOT NULL,
    upgrade_recipe_id integer NOT NULL,
    disassemble_recipe_id integer NOT NULL
);
CREATE SEQUENCE public.recipe_associations_association_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.recipe_associations_association_id_seq OWNED BY public.recipe_associations.association_id;
CREATE TABLE public.recipe_unlocks (
    user_id uuid NOT NULL,
    recipe_id integer NOT NULL,
    unlocked_at timestamp without time zone DEFAULT now()
);
CREATE TABLE public.stats_aggregates (
    aggregate_id integer NOT NULL,
    period character varying(20) NOT NULL,
    period_start timestamp without time zone NOT NULL,
    period_end timestamp without time zone NOT NULL,
    metrics jsonb NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);
CREATE SEQUENCE public.stats_aggregates_aggregate_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.stats_aggregates_aggregate_id_seq OWNED BY public.stats_aggregates.aggregate_id;
CREATE TABLE public.stats_events (
    event_id bigint NOT NULL,
    user_id uuid,
    event_type character varying(100) NOT NULL,
    event_data jsonb,
    created_at timestamp without time zone DEFAULT now()
);
CREATE SEQUENCE public.stats_events_event_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;
ALTER SEQUENCE public.stats_events_event_id_seq OWNED BY public.stats_events.event_id;
CREATE TABLE public.user_cooldowns (
    user_id uuid NOT NULL,
    action_name character varying(50) NOT NULL,
    last_used_at timestamp with time zone DEFAULT now() NOT NULL
);
CREATE TABLE public.user_inventory (
    user_id uuid NOT NULL,
    inventory_data jsonb DEFAULT '{"slots": []}'::jsonb
);
CREATE TABLE public.user_jobs (
    user_id uuid NOT NULL,
    job_id integer NOT NULL,
    current_xp bigint DEFAULT 0 NOT NULL,
    current_level integer DEFAULT 0 NOT NULL,
    xp_gained_today bigint DEFAULT 0,
    last_xp_gain timestamp with time zone
);
CREATE TABLE public.user_platform_links (
    user_id uuid NOT NULL,
    platform_id integer NOT NULL,
    platform_user_id character varying(255) NOT NULL
);
CREATE TABLE public.user_progression (
    user_id character varying(255) NOT NULL,
    progression_type character varying(50) NOT NULL,
    progression_key character varying(100) NOT NULL,
    unlocked_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    metadata jsonb
);
CREATE TABLE public.user_votes (
    user_id character varying(255) NOT NULL,
    node_id integer NOT NULL,
    target_level integer DEFAULT 1 NOT NULL,
    voted_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP,
    session_id integer,
    option_id integer
);
CREATE TABLE public.users (
    user_id uuid DEFAULT gen_random_uuid() NOT NULL,
    username character varying(255) NOT NULL,
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now()
);
ALTER TABLE ONLY public.crafting_recipes ALTER COLUMN recipe_id SET DEFAULT nextval('public.crafting_recipes_recipe_id_seq'::regclass);
ALTER TABLE ONLY public.disassemble_outputs ALTER COLUMN output_id SET DEFAULT nextval('public.disassemble_outputs_output_id_seq'::regclass);
ALTER TABLE ONLY public.disassemble_recipes ALTER COLUMN recipe_id SET DEFAULT nextval('public.disassemble_recipes_recipe_id_seq'::regclass);
ALTER TABLE ONLY public.engagement_metrics ALTER COLUMN id SET DEFAULT nextval('public.engagement_metrics_id_seq'::regclass);
ALTER TABLE ONLY public.events ALTER COLUMN id SET DEFAULT nextval('public.events_id_seq'::regclass);
ALTER TABLE ONLY public.item_types ALTER COLUMN item_type_id SET DEFAULT nextval('public.item_types_item_type_id_seq'::regclass);
ALTER TABLE ONLY public.items ALTER COLUMN item_id SET DEFAULT nextval('public.items_item_id_seq'::regclass);
ALTER TABLE ONLY public.job_level_bonuses ALTER COLUMN id SET DEFAULT nextval('public.job_level_bonuses_id_seq'::regclass);
ALTER TABLE ONLY public.jobs ALTER COLUMN id SET DEFAULT nextval('public.jobs_id_seq'::regclass);
ALTER TABLE ONLY public.platforms ALTER COLUMN platform_id SET DEFAULT nextval('public.platforms_platform_id_seq'::regclass);
ALTER TABLE ONLY public.progression_nodes ALTER COLUMN id SET DEFAULT nextval('public.progression_nodes_id_seq'::regclass);
ALTER TABLE ONLY public.progression_resets ALTER COLUMN id SET DEFAULT nextval('public.progression_resets_id_seq'::regclass);
ALTER TABLE ONLY public.progression_unlock_progress ALTER COLUMN id SET DEFAULT nextval('public.progression_unlock_progress_id_seq'::regclass);
ALTER TABLE ONLY public.progression_unlocks ALTER COLUMN id SET DEFAULT nextval('public.progression_unlocks_id_seq'::regclass);
ALTER TABLE ONLY public.progression_voting ALTER COLUMN id SET DEFAULT nextval('public.progression_voting_id_seq'::regclass);
ALTER TABLE ONLY public.progression_voting_options ALTER COLUMN id SET DEFAULT nextval('public.progression_voting_options_id_seq'::regclass);
ALTER TABLE ONLY public.progression_voting_sessions ALTER COLUMN id SET DEFAULT nextval('public.progression_voting_sessions_id_seq'::regclass);
ALTER TABLE ONLY public.recipe_associations ALTER COLUMN association_id SET DEFAULT nextval('public.recipe_associations_association_id_seq'::regclass);
ALTER TABLE ONLY public.stats_aggregates ALTER COLUMN aggregate_id SET DEFAULT nextval('public.stats_aggregates_aggregate_id_seq'::regclass);
ALTER TABLE ONLY public.stats_events ALTER COLUMN event_id SET DEFAULT nextval('public.stats_events_event_id_seq'::regclass);
ALTER TABLE ONLY public.crafting_recipes
    ADD CONSTRAINT crafting_recipes_pkey PRIMARY KEY (recipe_id);
ALTER TABLE ONLY public.crafting_recipes
    ADD CONSTRAINT crafting_recipes_target_item_id_key UNIQUE (target_item_id);
ALTER TABLE ONLY public.disassemble_outputs
    ADD CONSTRAINT disassemble_outputs_pkey PRIMARY KEY (output_id);
ALTER TABLE ONLY public.disassemble_recipes
    ADD CONSTRAINT disassemble_recipes_pkey PRIMARY KEY (recipe_id);
ALTER TABLE ONLY public.engagement_metrics
    ADD CONSTRAINT engagement_metrics_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.engagement_weights
    ADD CONSTRAINT engagement_weights_pkey PRIMARY KEY (metric_type);
ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.gamble_opened_items
    ADD CONSTRAINT gamble_opened_items_gamble_id_user_id_item_id_key UNIQUE (gamble_id, user_id, item_id);
ALTER TABLE ONLY public.gamble_participants
    ADD CONSTRAINT gamble_participants_pkey PRIMARY KEY (gamble_id, user_id);
ALTER TABLE ONLY public.gambles
    ADD CONSTRAINT gambles_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.item_type_assignments
    ADD CONSTRAINT item_type_assignments_pkey PRIMARY KEY (item_id, item_type_id);
ALTER TABLE ONLY public.item_types
    ADD CONSTRAINT item_types_pkey PRIMARY KEY (item_type_id);
ALTER TABLE ONLY public.item_types
    ADD CONSTRAINT item_types_type_name_key UNIQUE (type_name);
ALTER TABLE ONLY public.items
    ADD CONSTRAINT items_internal_name_key UNIQUE (internal_name);
ALTER TABLE ONLY public.items
    ADD CONSTRAINT items_pkey PRIMARY KEY (item_id);
ALTER TABLE ONLY public.job_level_bonuses
    ADD CONSTRAINT job_level_bonuses_job_id_min_level_bonus_type_key UNIQUE (job_id, min_level, bonus_type);
ALTER TABLE ONLY public.job_level_bonuses
    ADD CONSTRAINT job_level_bonuses_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.job_xp_events
    ADD CONSTRAINT job_xp_events_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_job_key_key UNIQUE (job_key);
ALTER TABLE ONLY public.jobs
    ADD CONSTRAINT jobs_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.link_tokens
    ADD CONSTRAINT link_tokens_pkey PRIMARY KEY (token);
ALTER TABLE ONLY public.moderators
    ADD CONSTRAINT moderators_pkey PRIMARY KEY (moderator_id);
ALTER TABLE ONLY public.platforms
    ADD CONSTRAINT platforms_name_key UNIQUE (name);
ALTER TABLE ONLY public.platforms
    ADD CONSTRAINT platforms_pkey PRIMARY KEY (platform_id);
ALTER TABLE ONLY public.progression_nodes
    ADD CONSTRAINT progression_nodes_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.progression_prerequisites
    ADD CONSTRAINT progression_prerequisites_pkey PRIMARY KEY (node_id, prerequisite_node_id);
ALTER TABLE ONLY public.progression_resets
    ADD CONSTRAINT progression_resets_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.progression_unlock_progress
    ADD CONSTRAINT progression_unlock_progress_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.progression_unlocks
    ADD CONSTRAINT progression_unlocks_node_id_current_level_key UNIQUE (node_id, current_level);
ALTER TABLE ONLY public.progression_unlocks
    ADD CONSTRAINT progression_unlocks_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.progression_voting
    ADD CONSTRAINT progression_voting_node_id_target_level_key UNIQUE (node_id, target_level);
ALTER TABLE ONLY public.progression_voting_options
    ADD CONSTRAINT progression_voting_options_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.progression_voting_options
    ADD CONSTRAINT progression_voting_options_session_id_node_id_target_level_key UNIQUE (session_id, node_id, target_level);
ALTER TABLE ONLY public.progression_voting
    ADD CONSTRAINT progression_voting_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.progression_voting_sessions
    ADD CONSTRAINT progression_voting_sessions_pkey PRIMARY KEY (id);
ALTER TABLE ONLY public.recipe_associations
    ADD CONSTRAINT recipe_associations_pkey PRIMARY KEY (association_id);
ALTER TABLE ONLY public.recipe_unlocks
    ADD CONSTRAINT recipe_unlocks_pkey PRIMARY KEY (user_id, recipe_id);
ALTER TABLE ONLY public.stats_aggregates
    ADD CONSTRAINT stats_aggregates_period_period_start_key UNIQUE (period, period_start);
ALTER TABLE ONLY public.stats_aggregates
    ADD CONSTRAINT stats_aggregates_pkey PRIMARY KEY (aggregate_id);
ALTER TABLE ONLY public.stats_events
    ADD CONSTRAINT stats_events_pkey PRIMARY KEY (event_id);
ALTER TABLE ONLY public.recipe_associations
    ADD CONSTRAINT unique_association UNIQUE (upgrade_recipe_id, disassemble_recipe_id);
ALTER TABLE ONLY public.disassemble_outputs
    ADD CONSTRAINT unique_recipe_output UNIQUE (recipe_id, item_id);
ALTER TABLE ONLY public.disassemble_recipes
    ADD CONSTRAINT unique_source_item UNIQUE (source_item_id);
ALTER TABLE ONLY public.user_cooldowns
    ADD CONSTRAINT user_cooldowns_pkey PRIMARY KEY (user_id, action_name);
ALTER TABLE ONLY public.user_inventory
    ADD CONSTRAINT user_inventory_pkey PRIMARY KEY (user_id);
ALTER TABLE ONLY public.user_jobs
    ADD CONSTRAINT user_jobs_pkey PRIMARY KEY (user_id, job_id);
ALTER TABLE ONLY public.user_platform_links
    ADD CONSTRAINT user_platform_links_pkey PRIMARY KEY (user_id, platform_id);
ALTER TABLE ONLY public.user_platform_links
    ADD CONSTRAINT user_platform_links_platform_id_platform_user_id_key UNIQUE (platform_id, platform_user_id);
ALTER TABLE ONLY public.user_progression
    ADD CONSTRAINT user_progression_pkey PRIMARY KEY (user_id, progression_type, progression_key);
ALTER TABLE ONLY public.user_votes
    ADD CONSTRAINT user_votes_pkey PRIMARY KEY (user_id, node_id, target_level);
ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (user_id);
CREATE INDEX idx_engagement_metrics_type_time ON public.engagement_metrics USING btree (metric_type, recorded_at);
CREATE INDEX idx_engagement_metrics_user ON public.engagement_metrics USING btree (user_id, metric_type);
CREATE INDEX idx_events_created ON public.events USING btree (created_at DESC);
CREATE INDEX idx_events_payload ON public.events USING gin (payload);
CREATE INDEX idx_events_type ON public.events USING btree (event_type);
CREATE INDEX idx_events_user ON public.events USING btree (user_id);
CREATE UNIQUE INDEX idx_gamble_participants_unique_user ON public.gamble_participants USING btree (gamble_id, user_id);
CREATE UNIQUE INDEX idx_gambles_single_active ON public.gambles USING btree (state) WHERE (state = ANY (ARRAY['Joining'::text, 'Opening'::text]));
CREATE INDEX idx_gambles_state ON public.gambles USING btree (state);
CREATE INDEX idx_goi_gamble_id ON public.gamble_opened_items USING btree (gamble_id);
CREATE INDEX idx_gp_gamble_id ON public.gamble_participants USING btree (gamble_id);
CREATE INDEX idx_inventory_item_id ON public.user_inventory USING gin (inventory_data);
CREATE UNIQUE INDEX idx_items_public_name ON public.items USING btree (public_name) WHERE (public_name IS NOT NULL);
CREATE INDEX idx_job_xp_events_job ON public.job_xp_events USING btree (job_id);
CREATE INDEX idx_job_xp_events_user ON public.job_xp_events USING btree (user_id, recorded_at DESC);
CREATE INDEX idx_link_tokens_expires ON public.link_tokens USING btree (expires_at);
CREATE INDEX idx_link_tokens_source ON public.link_tokens USING btree (source_platform, source_platform_id);
CREATE INDEX idx_link_tokens_state ON public.link_tokens USING btree (state);
CREATE INDEX idx_platform_user_id ON public.user_platform_links USING btree (platform_user_id);
CREATE INDEX idx_progression_prerequisites_prerequisite ON public.progression_prerequisites USING btree (prerequisite_node_id);
CREATE INDEX idx_progression_unlocks_node ON public.progression_unlocks USING btree (node_id);
CREATE INDEX idx_recipe_unlocks_user ON public.recipe_unlocks USING btree (user_id);
CREATE INDEX idx_recipes_target_item ON public.crafting_recipes USING btree (target_item_id);
CREATE INDEX idx_stats_aggregates_period ON public.stats_aggregates USING btree (period, period_start);
CREATE INDEX idx_stats_events_created_at ON public.stats_events USING btree (created_at);
CREATE INDEX idx_stats_events_event_type ON public.stats_events USING btree (event_type);
CREATE INDEX idx_stats_events_user_id ON public.stats_events USING btree (user_id);
CREATE INDEX idx_stats_events_user_type ON public.stats_events USING btree (user_id, event_type);
CREATE INDEX idx_unlock_progress_active ON public.progression_unlock_progress USING btree (unlocked_at) WHERE (unlocked_at IS NULL);
CREATE INDEX idx_user_cooldowns_user_action ON public.user_cooldowns USING btree (user_id, action_name);
CREATE INDEX idx_user_jobs_level ON public.user_jobs USING btree (current_level DESC);
CREATE INDEX idx_user_jobs_user ON public.user_jobs USING btree (user_id);
CREATE INDEX idx_user_progression ON public.user_progression USING btree (user_id, progression_type);
CREATE INDEX idx_user_votes_session ON public.user_votes USING btree (session_id) WHERE (session_id IS NOT NULL);
CREATE INDEX idx_voting_active ON public.progression_voting USING btree (is_active, voting_ends_at);
CREATE INDEX idx_voting_options_session ON public.progression_voting_options USING btree (session_id);
CREATE INDEX idx_voting_sessions_active ON public.progression_voting_sessions USING btree (status) WHERE ((status)::text = 'voting'::text);
CREATE INDEX idx_voting_sessions_status ON public.progression_voting_sessions USING btree (status);
ALTER TABLE ONLY public.crafting_recipes
    ADD CONSTRAINT crafting_recipes_target_item_id_fkey FOREIGN KEY (target_item_id) REFERENCES public.items(item_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.disassemble_outputs
    ADD CONSTRAINT disassemble_outputs_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(item_id);
ALTER TABLE ONLY public.disassemble_outputs
    ADD CONSTRAINT disassemble_outputs_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.disassemble_recipes(recipe_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.disassemble_recipes
    ADD CONSTRAINT disassemble_recipes_source_item_id_fkey FOREIGN KEY (source_item_id) REFERENCES public.items(item_id);
ALTER TABLE ONLY public.progression_voting_sessions
    ADD CONSTRAINT fk_winning_option FOREIGN KEY (winning_option_id) REFERENCES public.progression_voting_options(id);
ALTER TABLE ONLY public.gamble_opened_items
    ADD CONSTRAINT gamble_opened_items_gamble_id_fkey FOREIGN KEY (gamble_id) REFERENCES public.gambles(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.gamble_opened_items
    ADD CONSTRAINT gamble_opened_items_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(item_id);
ALTER TABLE ONLY public.gamble_opened_items
    ADD CONSTRAINT gamble_opened_items_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);
ALTER TABLE ONLY public.gamble_participants
    ADD CONSTRAINT gamble_participants_gamble_id_fkey FOREIGN KEY (gamble_id) REFERENCES public.gambles(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.gamble_participants
    ADD CONSTRAINT gamble_participants_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);
ALTER TABLE ONLY public.gambles
    ADD CONSTRAINT gambles_initiator_id_fkey FOREIGN KEY (initiator_id) REFERENCES public.users(user_id);
ALTER TABLE ONLY public.item_type_assignments
    ADD CONSTRAINT item_type_assignments_item_id_fkey FOREIGN KEY (item_id) REFERENCES public.items(item_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.item_type_assignments
    ADD CONSTRAINT item_type_assignments_item_type_id_fkey FOREIGN KEY (item_type_id) REFERENCES public.item_types(item_type_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.job_level_bonuses
    ADD CONSTRAINT job_level_bonuses_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id);
ALTER TABLE ONLY public.job_xp_events
    ADD CONSTRAINT job_xp_events_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id);
ALTER TABLE ONLY public.job_xp_events
    ADD CONSTRAINT job_xp_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id);
ALTER TABLE ONLY public.moderators
    ADD CONSTRAINT moderators_moderator_id_fkey FOREIGN KEY (moderator_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.progression_prerequisites
    ADD CONSTRAINT progression_prerequisites_node_id_fkey FOREIGN KEY (node_id) REFERENCES public.progression_nodes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.progression_prerequisites
    ADD CONSTRAINT progression_prerequisites_prerequisite_node_id_fkey FOREIGN KEY (prerequisite_node_id) REFERENCES public.progression_nodes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.progression_unlock_progress
    ADD CONSTRAINT progression_unlock_progress_node_id_fkey FOREIGN KEY (node_id) REFERENCES public.progression_nodes(id);
ALTER TABLE ONLY public.progression_unlock_progress
    ADD CONSTRAINT progression_unlock_progress_voting_session_id_fkey FOREIGN KEY (voting_session_id) REFERENCES public.progression_voting_sessions(id);
ALTER TABLE ONLY public.progression_unlocks
    ADD CONSTRAINT progression_unlocks_node_id_fkey FOREIGN KEY (node_id) REFERENCES public.progression_nodes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.progression_voting
    ADD CONSTRAINT progression_voting_node_id_fkey FOREIGN KEY (node_id) REFERENCES public.progression_nodes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.progression_voting_options
    ADD CONSTRAINT progression_voting_options_node_id_fkey FOREIGN KEY (node_id) REFERENCES public.progression_nodes(id);
ALTER TABLE ONLY public.progression_voting_options
    ADD CONSTRAINT progression_voting_options_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.progression_voting_sessions(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_associations
    ADD CONSTRAINT recipe_associations_disassemble_recipe_id_fkey FOREIGN KEY (disassemble_recipe_id) REFERENCES public.disassemble_recipes(recipe_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_associations
    ADD CONSTRAINT recipe_associations_upgrade_recipe_id_fkey FOREIGN KEY (upgrade_recipe_id) REFERENCES public.crafting_recipes(recipe_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_unlocks
    ADD CONSTRAINT recipe_unlocks_recipe_id_fkey FOREIGN KEY (recipe_id) REFERENCES public.crafting_recipes(recipe_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.recipe_unlocks
    ADD CONSTRAINT recipe_unlocks_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.stats_events
    ADD CONSTRAINT stats_events_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_cooldowns
    ADD CONSTRAINT user_cooldowns_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_inventory
    ADD CONSTRAINT user_inventory_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_jobs
    ADD CONSTRAINT user_jobs_job_id_fkey FOREIGN KEY (job_id) REFERENCES public.jobs(id);
ALTER TABLE ONLY public.user_jobs
    ADD CONSTRAINT user_jobs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_platform_links
    ADD CONSTRAINT user_platform_links_platform_id_fkey FOREIGN KEY (platform_id) REFERENCES public.platforms(platform_id);
ALTER TABLE ONLY public.user_platform_links
    ADD CONSTRAINT user_platform_links_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(user_id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_votes
    ADD CONSTRAINT user_votes_node_id_fkey FOREIGN KEY (node_id) REFERENCES public.progression_nodes(id) ON DELETE CASCADE;
ALTER TABLE ONLY public.user_votes
    ADD CONSTRAINT user_votes_option_id_fkey FOREIGN KEY (option_id) REFERENCES public.progression_voting_options(id);
ALTER TABLE ONLY public.user_votes
    ADD CONSTRAINT user_votes_session_id_fkey FOREIGN KEY (session_id) REFERENCES public.progression_voting_sessions(id);


-- Seed data from migrations 0005-0008, 0012
INSERT INTO item_types (type_name) VALUES ('consumable'), ('upgradeable') ON CONFLICT DO NOTHING;
INSERT INTO items (internal_name, public_name, item_description, base_value, default_display) VALUES 
    ('lootbox_tier0', 'junkbox', 'A basic lootbox containing random items', 100, 'Rusty Lootbox'),
    ('lootbox_tier1', 'lootbox', 'An upgraded lootbox with better rewards', 500, 'Basic Lootbox'),
    ('lootbox_tier2', 'goldbox', 'A premium lootbox with rare items', 2500, 'Golden Lootbox');
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, it.item_type_id
FROM items i
CROSS JOIN item_types it
WHERE i.internal_name IN ('lootbox_tier0', 'lootbox_tier1', 'lootbox_tier2')
  AND it.type_name = 'consumable';

INSERT INTO item_types (type_name) VALUES ('sellable'), ('buyable') ON CONFLICT DO NOTHING;
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, it.item_type_id
FROM items i
CROSS JOIN item_types it
WHERE i.internal_name IN ('lootbox_tier0', 'lootbox_tier1', 'lootbox_tier2')
  AND it.type_name IN ('sellable', 'buyable');

INSERT INTO items (internal_name, public_name, item_description, base_value, default_display) VALUES 
    ('money', 'money', 'Currency used for transactions', 1, 'Coins')
ON CONFLICT DO NOTHING;

INSERT INTO items (internal_name, public_name, item_description, base_value, default_display) VALUES 
    ('weapon_blaster', 'missile', 'A powerful weapon', 1000, 'Ray Gun')
ON CONFLICT DO NOTHING;
INSERT INTO item_type_assignments (item_id, item_type_id)
SELECT i.item_id, it.item_type_id
FROM items i
CROSS JOIN item_types it
WHERE i.internal_name = 'weapon_blaster'
  AND it.type_name IN ('upgradeable', 'consumable');

INSERT INTO platforms (name) VALUES ('twitch'), ('youtube'), ('discord') ON CONFLICT (name) DO NOTHING;

-- Progression nodes (minimal seed for tests - full tree synced from JSON config at runtime)
INSERT INTO progression_nodes (node_key, node_type, display_name, description, tier, size, category, unlock_cost, max_level, sort_order)
VALUES ('progression_system', 'feature', 'Progression System', 'The starting point of progression', 1, 'medium', 'core', 0, 1, 0)
ON CONFLICT DO NOTHING;

-- Auto-unlock the root progression node
INSERT INTO progression_unlocks (node_id, current_level, unlocked_by, engagement_score)
SELECT id, 1, 'auto', 0
FROM progression_nodes
WHERE node_key = 'progression_system'
ON CONFLICT DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS user_votes CASCADE;
DROP TABLE IF EXISTS progression_voting_options CASCADE;
DROP TABLE IF EXISTS progression_voting_sessions CASCADE;
DROP TABLE IF EXISTS progression_voting CASCADE;
DROP TABLE IF EXISTS progression_unlock_progress CASCADE;
DROP TABLE IF EXISTS progression_unlocks CASCADE;
DROP TABLE IF EXISTS progression_prerequisites CASCADE;
DROP TABLE IF EXISTS progression_nodes CASCADE;
DROP TABLE IF EXISTS progression_resets CASCADE;
DROP TABLE IF EXISTS user_progression CASCADE;
DROP TABLE IF EXISTS engagement_weights CASCADE;
DROP TABLE IF EXISTS engagement_metrics CASCADE;
DROP TABLE IF EXISTS user_jobs CASCADE;
DROP TABLE IF EXISTS job_xp_events CASCADE;
DROP TABLE IF EXISTS job_level_bonuses CASCADE;
DROP TABLE IF EXISTS jobs CASCADE;
DROP TABLE IF EXISTS gamble_opened_items CASCADE;
DROP TABLE IF EXISTS gamble_participants CASCADE;
DROP TABLE IF EXISTS gambles CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS user_cooldowns CASCADE;
DROP TABLE IF EXISTS recipe_unlocks CASCADE;
DROP TABLE IF EXISTS recipe_associations CASCADE;
DROP TABLE IF EXISTS disassemble_outputs CASCADE;
DROP TABLE IF EXISTS disassemble_recipes CASCADE;
DROP TABLE IF EXISTS crafting_recipes CASCADE;
DROP TABLE IF EXISTS item_type_assignments CASCADE;
DROP TABLE IF EXISTS item_types CASCADE;
DROP TABLE IF EXISTS user_inventory CASCADE;
DROP TABLE IF EXISTS items CASCADE;
DROP TABLE IF EXISTS link_tokens CASCADE;
DROP TABLE IF EXISTS user_platform_links CASCADE;
DROP TABLE IF EXISTS platforms CASCADE;
DROP TABLE IF EXISTS stats_events CASCADE;
DROP TABLE IF EXISTS stats_aggregates CASCADE;
DROP TABLE IF EXISTS moderators CASCADE;
DROP TABLE IF EXISTS users CASCADE;

DROP EXTENSION IF EXISTS "uuid-ossp";
