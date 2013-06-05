package catcher

import "sort"

type PodcastsBy func(p1, p2 * PodFeed) bool

func (by PodcastsBy) sort(podcasts []PodFeed) {
	ps := &podcastSorter{
		podcasts: podcasts,
		by : by,
	}
	sort.Sort(ps)
}

type podcastSorter struct {
	podcasts []PodFeed
	by func(p1, p2 * PodFeed) bool
}

func (s * podcastSorter) Len() int {
	return len(s.podcasts)
}

func (s * podcastSorter) Swap(i, j int) {
	s.podcasts[i], s.podcasts[j] = s.podcasts[j], s.podcasts[i]
}

func (s * podcastSorter) Less(i, j int) bool {
	return s.by(&s.podcasts[i], &s.podcasts[j])
}

func (catcher * Catcher) SortPodcastsByName() {
	nameSorter := func(p1, p2 * PodFeed) bool {
		return p1.Name < p2.Name
	}
	PodcastsBy(nameSorter).sort(catcher.Podcasts)
}

type EpisodesBy func(p1, p2 * PodEpisode) bool

func (by EpisodesBy) sort(episodes []PodEpisode) {
	ps := &episodeSorter{
		episodes: episodes,
		by : by,
	}
	sort.Sort(ps)
}

type episodeSorter struct {
	episodes []PodEpisode
	by func(p1, p2 * PodEpisode) bool
}

func (s * episodeSorter) Len() int {
	return len(s.episodes)
}

func (s * episodeSorter) Swap(i, j int) {
	s.episodes[i], s.episodes[j] = s.episodes[j], s.episodes[i]
}

func (s * episodeSorter) Less(i, j int) bool {
	return s.by(&s.episodes[i], &s.episodes[j])
}

func (feed * PodFeed) SortEpisodesByDate() {
	dateSorter := func(p1, p2 * PodEpisode) bool {
		return p1.ReleaseDate().Before(p2.ReleaseDate())
	}
	EpisodesBy(dateSorter).sort(feed.PodcastEpisodes)
}