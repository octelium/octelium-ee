const Meta = (props: { title?: string }) => {
  return (
    <title>
      {props.title ? `${props.title} - Octelium Console` : `Octelium Console`}
    </title>
  );
};

export default Meta;
